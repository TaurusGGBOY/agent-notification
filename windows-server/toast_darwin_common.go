//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Foundation -framework Cocoa -framework UserNotifications
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#import <objc/runtime.h>
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>

static NSString *agentnotify_string(const char *value) {
	if (value == NULL) {
		return @"";
	}

	NSString *string = [NSString stringWithUTF8String:value];
	if (string != nil) {
		return string;
	}

	size_t length = strlen(value);
	string = [[NSString alloc] initWithBytes:value length:length encoding:NSUTF8StringEncoding];
	if (string != nil) {
		return string;
	}

	string = [[NSString alloc] initWithBytes:value length:length encoding:NSISOLatin1StringEncoding];
	if (string != nil) {
		return string;
	}

	return @"";
}

static NSString *agentnotifyFallbackBundleIdentifier = @"com.agentnotify.client";

@interface NSBundle (AgentNotifyBundleIdentifier)
- (NSString *)agentnotify_bundleIdentifier;
@end

@implementation NSBundle (AgentNotifyBundleIdentifier)
- (NSString *)agentnotify_bundleIdentifier {
	NSString *identifier = [self agentnotify_bundleIdentifier];
	if (identifier != nil && [identifier length] > 0) {
		return identifier;
	}
	if (self == [NSBundle mainBundle]) {
		return agentnotifyFallbackBundleIdentifier;
	}
	return identifier;
}
@end

@interface AgentNotifyNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation AgentNotifyNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	UNNotificationPresentationOptions options = UNNotificationPresentationOptionSound;
	if (@available(macOS 11.0, *)) {
		options |= UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList;
	} else {
		options |= UNNotificationPresentationOptionAlert;
	}
	completionHandler(options);
}
@end

static AgentNotifyNotificationDelegate *agentnotifyNotificationDelegate = nil;

static void agentnotify_ensure_bundle_identifier(const char *bundleIdentifier) {
	NSString *fallbackIdentifier = agentnotify_string(bundleIdentifier);
	if ([fallbackIdentifier length] > 0) {
		agentnotifyFallbackBundleIdentifier = [fallbackIdentifier copy];
	}

	if ([[[NSBundle mainBundle] bundleIdentifier] length] > 0) {
		return;
	}

	static dispatch_once_t once;
	dispatch_once(&once, ^{
		Method original = class_getInstanceMethod([NSBundle class], @selector(bundleIdentifier));
		Method replacement = class_getInstanceMethod([NSBundle class], @selector(agentnotify_bundleIdentifier));
		if (original != NULL && replacement != NULL) {
			method_exchangeImplementations(original, replacement);
		}
	});
}

static void agentnotify_configure_notification_center(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		agentnotifyNotificationDelegate = [[AgentNotifyNotificationDelegate alloc] init];
		[[UNUserNotificationCenter currentNotificationCenter] setDelegate:agentnotifyNotificationDelegate];
	});
}

// Initialize NSApplication as a background accessory (no Dock icon).
// Must be called once before any notification API usage.
static void agentnotify_init_app(const char *bundleIdentifier) {
	agentnotify_ensure_bundle_identifier(bundleIdentifier);
	[NSApplication sharedApplication];
	[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
	agentnotify_configure_notification_center();
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
			__block NSError *authError = nil;
			dispatch_semaphore_t authSemaphore = dispatch_semaphore_create(0);
			[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
				completionHandler:^(BOOL granted, NSError *error) {
					authorized = granted;
					authError = error;
					dispatch_semaphore_signal(authSemaphore);
				}];

			if (dispatch_semaphore_wait(authSemaphore, dispatch_time(DISPATCH_TIME_NOW, 5 * NSEC_PER_SEC)) != 0) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "notification authorization timed out");
				}
				return 1;
			}

			if (authError != nil) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "%s", [[authError localizedDescription] UTF8String]);
				}
				return 1;
			}

			if (!authorized) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "notification authorization not granted");
				}
				return 1;
			}

			// Build notification content.
			UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
			content.title = agentnotify_string(title);
			content.subtitle = agentnotify_string(subtitle);
			content.body = agentnotify_string(body);
			content.sound = [UNNotificationSound defaultSound];

			// Create an immediate trigger (timeInterval must be > 0).
			UNTimeIntervalNotificationTrigger *trigger =
				[UNTimeIntervalNotificationTrigger triggerWithTimeInterval:0.1 repeats:NO];

			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:
				[[NSUUID UUID] UUIDString]
				content:content trigger:trigger];

			// Synchronously add the notification request.
			__block NSError *addError = nil;
			dispatch_semaphore_t addSemaphore = dispatch_semaphore_create(0);
			[center addNotificationRequest:request withCompletionHandler:^(NSError *error) {
				addError = error;
				dispatch_semaphore_signal(addSemaphore);
			}];

			if (dispatch_semaphore_wait(addSemaphore, dispatch_time(DISPATCH_TIME_NOW, 3 * NSEC_PER_SEC)) != 0) {
				if (errbuf != NULL && errbuflen > 0) {
					snprintf(errbuf, errbuflen, "notification delivery timed out");
				}
				return 1;
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
		bundleID := C.CString(darwinNotificationBundleIdentifier)
		defer C.free(unsafe.Pointer(bundleID))
		C.agentnotify_init_app(bundleID)
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
