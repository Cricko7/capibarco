import 'package:capibarco/app/app.dart';
import 'package:capibarco/bootstrap/providers.dart';
import 'package:capibarco/core/analytics/analytics_service.dart';
import 'package:capibarco/core/cache/in_memory_json_cache_store.dart';
import 'package:capibarco/core/config/environment.dart';
import 'package:capibarco/core/notifications/push_notifications_service.dart';
import 'package:capibarco/core/storage/in_memory_token_storage.dart';
import 'package:capibarco/features/auth/domain/entities/auth_session.dart';
import 'package:capibarco/features/auth/domain/repositories/auth_repository.dart';
import 'package:capibarco/features/auth/presentation/auth_controller.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthRepository implements AuthRepository {
  @override
  Future<void> clearSession() async {}

  @override
  Future<AuthSession> login({
    required String email,
    required String password,
    required String locale,
  }) async {
    return _session(email);
  }

  @override
  Future<AuthSession> refreshSession(String refreshToken) async =>
      _session('user@example.com');

  @override
  Future<AuthSession> register({
    required String email,
    required String password,
    required String locale,
  }) async {
    return _session(email);
  }

  @override
  Future<AuthSession?> restoreSession() async => null;

  AuthSession _session(String email) {
    return AuthSession(
      user: AuthUser(
        id: 'profile-1',
        tenantId: 'default',
        email: email,
        isActive: true,
      ),
      accessToken: 'token',
      refreshToken: 'refresh',
      expiresAt: DateTime.now().toUtc().add(const Duration(minutes: 15)),
    );
  }
}

void main() {
  testWidgets('renders login shell when unauthenticated', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          appEnvironmentProvider.overrideWithValue(
            AppEnvironment.fromEnvironment(),
          ),
          tokenStorageProvider.overrideWithValue(InMemoryTokenStorage()),
          cacheStoreProvider.overrideWithValue(InMemoryJsonCacheStore()),
          analyticsServiceProvider.overrideWithValue(
            const NoopAnalyticsService(),
          ),
          pushNotificationsServiceProvider.overrideWithValue(
            const NoopPushNotificationsService(),
          ),
          authRepositoryProvider.overrideWithValue(_FakeAuthRepository()),
        ],
        child: const CapibarcoApp(),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Sign in'), findsWidgets);
  });
}
