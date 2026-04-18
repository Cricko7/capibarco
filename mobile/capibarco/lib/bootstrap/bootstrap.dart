import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../app/app.dart';
import '../core/analytics/analytics_service.dart';
import '../core/cache/shared_preferences_cache_store.dart';
import '../core/config/environment.dart';
import '../core/notifications/push_notifications_service.dart';
import '../core/storage/secure_token_storage.dart';
import 'providers.dart';

Future<Widget> bootstrap() async {
  final environment = AppEnvironment.fromEnvironment();
  final sharedPreferences = await SharedPreferences.getInstance();
  final tokenStorage = SecureTokenStorage();
  final cacheStore = SharedPreferencesCacheStore(sharedPreferences);
  const analyticsService = NoopAnalyticsService();
  const pushNotificationsService = NoopPushNotificationsService();

  await analyticsService.initialize();
  await pushNotificationsService.initialize();

  return ProviderScope(
    overrides: [
      appEnvironmentProvider.overrideWithValue(environment),
      tokenStorageProvider.overrideWithValue(tokenStorage),
      cacheStoreProvider.overrideWithValue(cacheStore),
      analyticsServiceProvider.overrideWithValue(analyticsService),
      pushNotificationsServiceProvider.overrideWithValue(
        pushNotificationsService,
      ),
    ],
    child: const CapibarcoApp(),
  );
}
