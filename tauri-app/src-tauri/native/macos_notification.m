#import <Cocoa/Cocoa.h>
#import <Foundation/Foundation.h>
#import <UserNotifications/UserNotifications.h>
#include <stdarg.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

static void agentnotify_debug_log(const char *format, ...) {
	const char *enabled = getenv("AGENT_NOTIFY_DEBUG_NOTIFICATIONS");
	if (enabled == NULL || strcmp(enabled, "1") != 0) {
		return;
	}

	va_list args;
	va_start(args, format);
	fprintf(stderr, "agentnotify notification debug: ");
	vfprintf(stderr, format, args);
	fprintf(stderr, "\n");
	va_end(args);
}

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

@interface AgentNotifyAppNotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation AgentNotifyAppNotificationDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
	agentnotify_debug_log("willPresent id=%s", [[notification.request identifier] UTF8String]);
	UNNotificationPresentationOptions options = UNNotificationPresentationOptionSound;
	if (@available(macOS 11.0, *)) {
		options |= UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList;
	} else {
		options |= UNNotificationPresentationOptionAlert;
	}
	completionHandler(options);
}
@end

static AgentNotifyAppNotificationDelegate *agentnotifyNotificationDelegate = nil;

void agentnotify_configure_notification_center_early(void) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		agentnotifyNotificationDelegate = [[AgentNotifyAppNotificationDelegate alloc] init];
		[[UNUserNotificationCenter currentNotificationCenter] setDelegate:agentnotifyNotificationDelegate];
		agentnotify_debug_log("notification center delegate configured (early)");
	});
}

static void agentnotify_configure_notification_center(void) {
	agentnotify_configure_notification_center_early();
}

static void agentnotify_log_notification_settings(UNUserNotificationCenter *center) {
	const char *enabled = getenv("AGENT_NOTIFY_DEBUG_NOTIFICATIONS");
	if (enabled == NULL || strcmp(enabled, "1") != 0) {
		return;
	}

	dispatch_semaphore_t settingsSemaphore = dispatch_semaphore_create(0);
	[center getNotificationSettingsWithCompletionHandler:^(UNNotificationSettings *settings) {
		agentnotify_debug_log(
			"settings auth=%ld alertSetting=%ld soundSetting=%ld alertStyle=%ld notificationCenter=%ld lockScreen=%ld",
			(long)settings.authorizationStatus,
			(long)settings.alertSetting,
			(long)settings.soundSetting,
			(long)settings.alertStyle,
			(long)settings.notificationCenterSetting,
			(long)settings.lockScreenSetting
		);
		dispatch_semaphore_signal(settingsSemaphore);
	}];
	dispatch_semaphore_wait(settingsSemaphore, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
}

static void agentnotify_log_delivered_notifications(UNUserNotificationCenter *center) {
	const char *enabled = getenv("AGENT_NOTIFY_DEBUG_NOTIFICATIONS");
	if (enabled == NULL || strcmp(enabled, "1") != 0) {
		return;
	}

	dispatch_semaphore_t deliveredSemaphore = dispatch_semaphore_create(0);
	[center getDeliveredNotificationsWithCompletionHandler:^(NSArray<UNNotification *> *notifications) {
		agentnotify_debug_log("delivered count=%lu", (unsigned long)[notifications count]);
		for (UNNotification *notification in notifications) {
			agentnotify_debug_log(
				"delivered id=%s title=%s body=%s",
				[[notification.request identifier] UTF8String],
				[[notification.request.content title] UTF8String],
				[[notification.request.content body] UTF8String]
			);
		}
		dispatch_semaphore_signal(deliveredSemaphore);
	}];
	dispatch_semaphore_wait(deliveredSemaphore, dispatch_time(DISPATCH_TIME_NOW, 2 * NSEC_PER_SEC));
}

int agentnotify_show_app_notification(
	const char *title,
	const char *body,
	char *errbuf,
	int errbuflen
) {
	@autoreleasepool {
		@try {
			agentnotify_debug_log("show requested title=%s body=%s", title == NULL ? "" : title, body == NULL ? "" : body);
			agentnotify_debug_log(
				"bundleIdentifier=%s active=%d activationPolicy=%ld",
				[[[NSBundle mainBundle] bundleIdentifier] UTF8String],
				[NSApp isActive],
				(long)[NSApp activationPolicy]
			);
			agentnotify_configure_notification_center();
			UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
			agentnotify_log_notification_settings(center);

			__block BOOL authorized = NO;
			__block NSError *authError = nil;
			dispatch_semaphore_t authSemaphore = dispatch_semaphore_create(0);
			[center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
				completionHandler:^(BOOL granted, NSError *error) {
					authorized = granted;
					authError = error;
					agentnotify_debug_log("authorization granted=%d error=%s", granted, error == nil ? "" : [[error localizedDescription] UTF8String]);
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

			UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
			content.title = agentnotify_string(title);
			content.body = agentnotify_string(body);
			content.sound = [UNNotificationSound defaultSound];

			UNNotificationRequest *request = [UNNotificationRequest requestWithIdentifier:
				[[NSUUID UUID] UUIDString]
				content:content trigger:nil];

			__block NSError *addError = nil;
			dispatch_semaphore_t addSemaphore = dispatch_semaphore_create(0);
			[center addNotificationRequest:request withCompletionHandler:^(NSError *error) {
				addError = error;
				agentnotify_debug_log("add request id=%s error=%s", [[request identifier] UTF8String], error == nil ? "" : [[error localizedDescription] UTF8String]);
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

			[[NSRunLoop currentRunLoop] runUntilDate:
				[NSDate dateWithTimeIntervalSinceNow:0.3]];
			agentnotify_log_delivered_notifications(center);

			return 0;
		} @catch (NSException *exception) {
			if (errbuf != NULL && errbuflen > 0) {
				snprintf(errbuf, errbuflen, "%s", [[exception reason] UTF8String]);
			}
			return 1;
		}
	}
}
