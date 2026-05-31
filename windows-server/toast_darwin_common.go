//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Foundation -framework Cocoa -framework UserNotifications
#include <stdlib.h>
#include <stdio.h>
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

// Initialize NSApplication as a background accessory (no Dock icon).
// Must be called once before any notification API usage.
static void agentnotify_init_app(void) {
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
}

static int agentnotify_deliver_notification(
	const char *title,
	const char *subtitle,
	const char *body,
	const char *sound,
	char *errbuf,
	int errbuflen
) {
	@autoreleasepool {
		@try {
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];

			// Request authorization (returns immediately if already granted/denied).
			__block BOOL authorized = NO;
			__block BOOL authDone = NO;
			[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
				completionHandler:^(BOOL granted, NSError *error) {
					authorized = granted;
					authDone = YES;
				}];

			// Spin the run loop to process the authorization callback.
			// On subsequent calls (already authorized/denied), this returns almost instantly.
			for (int i = 0; i < 100 && !authDone; i++) {
				[[NSRunLoop currentRunLoop] runUntilDate:
					[NSDate dateWithTimeIntervalSinceNow:0.05]];
			}

			if (!authorized) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "notification authorization not granted");
				}
				return 1;
			}

			// Build notification content.
			UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
			content.title = [NSString stringWithUTF8String:title ? title : ""];
			content.subtitle = [NSString stringWithUTF8String:subtitle ? subtitle : ""];
			content.body = [NSString stringWithUTF8String:body ? body : ""];

			NSString *soundName = [NSString stringWithUTF8String:sound ? sound : ""];
			if ([soundName length] > 0) {
				content.sound = [UNNotificationSound soundNamed:soundName];
			} else {
				content.sound = [UNNotificationSound defaultSound];
			}

			// Create an immediate trigger (timeInterval must be > 0).
			UNTimeIntervalNotificationTrigger *trigger =
				[UNTimeIntervalNotificationTrigger triggerWithTimeInterval:0.1 repeats:NO];

			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:
				[[NSUUID UUID] UUIDString]
				content:content trigger:trigger];

			// Synchronously add the notification request.
			__block BOOL addDone = NO;
			__block NSError *addError = nil;
			[center addNotificationRequest:request withCompletionHandler:^(NSError *error) {
				addError = error;
				addDone = YES;
			}];

			// Spin the run loop to process the add-notification callback.
			for (int i = 0; i < 60 && !addDone; i++) {
				[[NSRunLoop currentRunLoop] runUntilDate:
					[NSDate dateWithTimeIntervalSinceNow:0.05]];
			}

			if (addError != nil) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "%s", [[addError localizedDescription] UTF8String]);
				}
				return 1;
			}

			// Brief additional spin so the system can render the notification.
			[[NSRunLoop currentRunLoop] runUntilDate:
				[NSDate dateWithTimeIntervalSinceNow:0.3]];

			return 0;
		} @catch (NSException *exception) {
			if (errbuf != NULL && errbuflen > 0) {
				snprintf(errbuf, errbuflen, "%s", [[exception reason] UTF8String]);
			}
			return 1;
		}
	}
}
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

var appOnce sync.Once

func ensureApp() {
	appOnce.Do(func() {
		C.agentnotify_init_app()
	})
}

func deliverDarwinNotification(req darwinNotificationRequest) error {
	ensureApp()

	title := C.CString(req.Title)
	subtitle := C.CString(req.Subtitle)
	body := C.CString(req.Body)
	sound := C.CString(req.Sound)
	errbuf := C.malloc(512)
	defer C.free(unsafe.Pointer(title))
	defer C.free(unsafe.Pointer(subtitle))
	defer C.free(unsafe.Pointer(body))
	defer C.free(unsafe.Pointer(sound))
	defer C.free(errbuf)

	result := C.agentnotify_deliver_notification(
		title,
		subtitle,
		body,
		sound,
		(*C.char)(errbuf),
		512,
	)
	if result != 0 {
		msg := C.GoString((*C.char)(errbuf))
		if strings.TrimSpace(msg) == "" {
			msg = "unknown native notification error"
		}
		return fmt.Errorf("native macOS notification: %s", msg)
	}
	return nil
}
