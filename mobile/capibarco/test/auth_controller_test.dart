import 'package:capibarco/features/auth/domain/entities/auth_session.dart';
import 'package:capibarco/features/auth/domain/repositories/auth_repository.dart';
import 'package:capibarco/features/auth/presentation/auth_controller.dart';
import 'package:capibarco/features/auth/presentation/auth_state.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthRepository implements AuthRepository {
  _FakeAuthRepository({this.storedSession, this.shouldFail = false});

  final AuthSession? storedSession;
  final bool shouldFail;

  @override
  Future<void> clearSession() async {}

  @override
  Future<AuthSession> login({
    required String email,
    required String password,
    required String locale,
  }) async {
    if (shouldFail) {
      throw Exception('login failed');
    }
    return _session(email);
  }

  @override
  Future<AuthSession> refreshSession(String refreshToken) async =>
      _session('stored@example.com');

  @override
  Future<AuthSession> register({
    required String email,
    required String password,
    required String locale,
  }) async {
    return _session(email);
  }

  @override
  Future<AuthSession?> restoreSession() async => storedSession;

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
  test('bootstrap restores session', () async {
    final container = ProviderContainer(
      overrides: [
        authRepositoryProvider.overrideWithValue(
          _FakeAuthRepository(
            storedSession: AuthSession(
              user: const AuthUser(
                id: 'profile-1',
                tenantId: 'default',
                email: 'stored@example.com',
                isActive: true,
              ),
              accessToken: 'token',
              refreshToken: 'refresh',
              expiresAt: DateTime.now().toUtc().add(
                const Duration(minutes: 15),
              ),
            ),
          ),
        ),
      ],
    );
    addTearDown(container.dispose);

    await container.read(authControllerProvider.notifier).bootstrap();

    final state = container.read(authControllerProvider);
    expect(state.status, AuthStatus.authenticated);
    expect(state.session?.user.email, 'stored@example.com');
  });

  test('login surfaces repository error', () async {
    final container = ProviderContainer(
      overrides: [
        authRepositoryProvider.overrideWithValue(
          _FakeAuthRepository(shouldFail: true),
        ),
      ],
    );
    addTearDown(container.dispose);

    await container
        .read(authControllerProvider.notifier)
        .login(
          email: 'user@example.com',
          password: 'password123',
          locale: 'en',
        );

    final state = container.read(authControllerProvider);
    expect(state.status, AuthStatus.unauthenticated);
    expect(state.errorMessage, contains('login failed'));
  });
}
