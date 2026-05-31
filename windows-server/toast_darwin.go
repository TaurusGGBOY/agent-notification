//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#include <stdlib.h>
#include <stdio.h>
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

@interface AgentNotifyNotificationDelegate : NSObject <NSUserNotificationCenterDelegate>
@end

@implementation AgentNotifyNotificationDelegate
- (BOOL)userNotificationCenter:(NSUserNotificationCenter *)center shouldPresentNotification:(NSUserNotification *)notification {
	return YES;
}
@end

static NSString *agentnotify_string(const char *value) {
	if (value == NULL) {
		return @"";
	}
	NSString *string = [NSString stringWithUTF8String:value];
	if (string == nil) {
		return @"";
	}
	return string;
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
			NSUserNotification *notification = [[NSUserNotification alloc] init];
			notification.title = agentnotify_string(title);
			notification.subtitle = agentnotify_string(subtitle);
			notification.informativeText = agentnotify_string(body);
			NSString *soundName = agentnotify_string(sound);
			if ([soundName length] > 0) {
				notification.soundName = soundName;
			} else {
				notification.soundName = NSUserNotificationDefaultSoundName;
			}

			NSUserNotificationCenter *center = [NSUserNotificationCenter defaultUserNotificationCenter];
			AgentNotifyNotificationDelegate *delegate = [[AgentNotifyNotificationDelegate alloc] init];
			center.delegate = delegate;
			[center deliverNotification:notification];
			[NSThread sleepForTimeInterval:0.8];
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
	"unsafe"
)

func deliverDarwinNotification(req darwinNotificationRequest) error {
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
