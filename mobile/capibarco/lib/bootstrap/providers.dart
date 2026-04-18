import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../core/analytics/analytics_service.dart';
import '../core/cache/json_cache_store.dart';
import '../core/config/environment.dart';
import '../core/notifications/push_notifications_service.dart';
import '../core/storage/token_storage.dart';

final appEnvironmentProvider = Provider<AppEnvironment>(
  (_) => throw UnimplementedError(
    'AppEnvironment must be overridden at bootstrap',
  ),
);

final tokenStorageProvider = Provider<TokenStorage>(
  (_) =>
      throw UnimplementedError('TokenStorage must be overridden at bootstrap'),
);

final cacheStoreProvider = Provider<JsonCacheStore>(
  (_) => throw UnimplementedError(
    'JsonCacheStore must be overridden at bootstrap',
  ),
);

final analyticsServiceProvider = Provider<AnalyticsService>(
  (_) => throw UnimplementedError(
    'AnalyticsService must be overridden at bootstrap',
  ),
);

final pushNotificationsServiceProvider = Provider<PushNotificationsService>(
  (_) => throw UnimplementedError(
    'PushNotificationsService must be overridden at bootstrap',
  ),
);
