# Calling Java from Go in a Gio UI Android App
[Abigail Adegbiji](https://aabiji.github.io/) • August 12, 2025

Lately I've been playing around with [Gio UI](https://gioui.org/), a UI
library for Go. Although Gio supports Android, getting it to interoperate
with Java code that uses the Android SDK turned out to be more complex
than expected due to a lack of solid resources on the topic. So I thought
I'd share what I managed to figure out. Here, I’ll walk through how to call
Java code from Go in an Android app.

Let's start with a basic Gio app that shows "Hello World!" on the screen:
```go
package main

import (
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/font/gofont"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		w := new(app.Window)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func loop(w *app.Window) error {
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			l := material.H1(th, "Hello world!")
			l.Alignment = text.Middle
			l.Layout(gtx)
			e.Frame(gtx.Ops)
		}
	}
}
```

To build it for Android, follow the official Gio Android setup
[guide](https://gioui.org/doc/install/android). Then run these commands:
```bash
# Runs any //go:generate directives (we’ll add some later for Java)
go generate

# The -ldflags "-checklinkname=0" flag is needed because of the
# [anet](https://github.com/wlynxg/anet) library Gio uses for networking.
gogio -ldflags "-checklinkname=0" -target android github.com/yourusername/yourmodule -o demo.apk

# Install to Android device
adb install demo.apk
```

Moving on to the Java side, we’ll create a method to write a file to the Android
Downloads folder using the Android SDK’s [MediaStore](https://developer.android.com/reference/kotlin/android/provider/MediaStore.html) API:
```java
package com.example.demo;

import android.content.ContentResolver;
import android.content.ContentValues;
import android.content.Context;
import android.net.Uri;
import android.os.Environment;
import android.provider.MediaStore;
import android.util.Log;

import java.io.OutputStream;

public class Utils {
    public static void writeToDownloadsFolder(
        Context context, byte[] contents, String filename, String mimetype) {
        try {
            ContentResolver resolver = context.getContentResolver();
            ContentValues values = new ContentValues();
            values.put(MediaStore.MediaColumns.DISPLAY_NAME, filename);
            values.put(MediaStore.MediaColumns.MIME_TYPE, mimetype);
            values.put(MediaStore.MediaColumns.RELATIVE_PATH,
                Environment.DIRECTORY_DOWNLOADS + "/");

            Uri uri = resolver.insert(MediaStore.Files.getContentUri("external"), values);
            OutputStream output = resolver.openOutputStream(uri);
            output.write(contents);
            output.flush();
            output.close();
        } catch (Exception e) {
            Log.e("demo-project", "Exception occurred!", e);
        }
    }
}
```

Now we need to compile the Java code into a jar and include it into our Gio app.
We can automatically do that by add `go:generate` directives at the top of the Go code:
```go
//go:generate javac -classpath $ANDROID_HOME/platforms/android-36/android.jar -d /tmp/java_classes Utils.java
//go:generate jar cf Utils.jar -C /tmp/java_classes .
```

Replace android-36 with your Android SDK version. These commands
compile Utils.java into class files, using the Android SDK’s android.jar for dependencies
and packages the class files into Utils.jar.

Now we get to the tricky part: calling our Java method from Go. Since
Android apps run in the Android Runtime (ART), a JVM compatible environment
and our app is written in Go, we need a way to bridge the two.
This is where the Java Native Interface (JNI) comes in.

The JNI enables native code (written in C/C++) to interact with Java code.
It operates through a set of functions exposed via a `JNIEnv` pointer,
which serves as the entry point for most JNI operations. These functions allow you to
locate java classes by name, create java objects from native data, invoke java methods,
handle exceptions and manage JVM state.

The Android Native Development Kit
([NDK](https://developer.android.com/training/articles/perf-jni))
implements those functions for the ART. We'll also be using [jnigi](https://github.com/timob/jnigi),
since it wraps the raw C JNI functions in nice Go abstractions.

The workflow for our Go code would be to get a `JNIEnv` pointer tied to the current thread
(since you can't share a JNIEnv between threads), convert Go arguments to the corresponding
Java objects, then locate the java class and methods and invoke it with arguments.

Given our example, it would look like this:
```go
import (
	"github.com/timob/jnigi"
	"gioui.org/app"
)

func writeToDownloadsFolder(filename, mimetype string, contents []byte) error {
	env, cleanup := getJNIEnv()
	defer cleanup()

	context := jnigi.WrapJObject(app.AppContext(), "android/content/Context", false)

	filenameObj, err := env.NewObject("java/lang/String", []byte(filename))
	if err != nil {
		return err
	}

	mimetypeObj, err := env.NewObject("java/lang/String", []byte(mimetype))
	if err != nil {
		return err
	}

	contentsObj := env.NewByteArrayFromSlice(contents)

	err = env.CallStaticMethod(
		"com/example/demo/Utils",
		"writeToDownloadsFolder",
		nil, // returns void
		context, contentsObj, filenameObj, mimetypeObj,
	)
	return err
}
```

Gio Android backend gives us the app's `Context`, which is a data structure that
gives the app access to Android ressources.

Now to get the `JNIEnv` pointer required for JNI operations, we'll use CGo to call
C functions from the Android NDK:
```go
/*
#cgo LDFLAGS: -llog -landroid

#include <android/log.h>
#include <android/native_window_jni.h>
#include <stdlib.h>

static jint jni_GetEnvOrAttach(JavaVM *vm, JNIEnv **env, jint *attached) {
    jint res = (*vm)->GetEnv(vm, (void **)env, JNI_VERSION_1_6);
    if (res == JNI_EDETACHED) {
        res = (*vm)->AttachCurrentThread(vm, (void **)env, NULL);
        *attached = res == JNI_OK;
    }
    return res;
}

static void jni_DetachCurrent(JavaVM *vm) {
    (*vm)->DetachCurrentThread(vm);
}
*/
import "C"
import (
    "runtime"
    "unsafe"
)

func getJNIEnv() (*jnigi.Env, func()) {
	runtime.LockOSThread()

	jvm := app.JavaVM()
	cJVM := (*C.JavaVM)(unsafe.Pointer(jvm))

	var cEnv *C.JNIEnv
	var attached C.jint
	C.jni_GetEnvOrAttach(cJVM, &cEnv, &attached)
	_, env := jnigi.UseJVM(unsafe.Pointer(jvm), unsafe.Pointer(cEnv), nil)

	cleanup := func() {
		if attached != 0 {
			C.jni_DetachCurrent(cJVM)
		}
		runtime.UnlockOSThread()
	}
	return env, cleanup
}
```

`jni_GetEnvOrAttach` uses the `JavaVM` pointer to query the current `JNIEnv` with `GetEnv`. 
If the current thread is not attached, it attaches it. This is necessary because JNI requires
threads to be explicitly attached to the JVM to obtain a valid `JNIEnv`.
We specify JNI_VERSION_1_6 for compatibility.

`jni_DetachCurrent` calls `DetachCurrentThread` to release the thread from the JVM,
preventing resource leaks.

In `getJNIEnv` we bind the goroutine to a specific OS thread, get the `JavaVM` pointer
from Gio's Android backend, call our C functions and define a cleanup function that
detaches the thread if it's attached and unlocks the OS thread. Note that the `nil`
when calling `jnigi.UseJVM` is for the optional thiz (`this` in Java), which we won't need
for static methods.

And that's that! For debugging purposes we'll add a panic handler:
```go
import (
	"fmt"
	"runtime"
    "runtime/debug"
	"strings"
	"unsafe"
)

func androidCrashHandler() {
	if r := recover(); r != nil {
        str := fmt.Sprintf("Crash: %v\n%s", r, debug.Stack())
        tag := C.CString("demo-project")
        defer C.free(unsafe.Pointer(tag))

        lines := strings.Split(str, "\n")
        for _, line := range lines {
            msg := C.CString(line)
            C.__android_log_write(C.ANDROID_LOG_INFO, tag, msg) // from <android/log.h>
            C.free(unsafe.Pointer(msg))
        }
	}
}
```

And update the main function to include the panic handler and test the Java call:
```go
func main() {
	if runtime.GOOS == "android" {
		defer androidCrashHandler()

        contents := []byte("Open your mind...")
        if err := writeToDownloadsFolder("this-works.txt", "text/plain", contents); err != nil {
            panic(fmt.Sprintf("Failed to write to Downloads: %s", err))
        }
	}

	go func() {
		w := new(app.Window)
		if err := loop(w); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}
```

Now, if you build and run the app, and check the Download folder in Internal Storage,
you should see `this-works.txt`. If not you can inspect the logs with
```bash
adb logcat -s demo-project
```

You can find the full code [here](https://gist.github.com/aabiji/254f27c987d99a58d1f4e949842e48d4).
