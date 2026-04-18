import 'package:capibarco/src/domain/entities/session.dart';
import 'package:capibarco/src/domain/repositories/auth_repository.dart';
import 'package:capibarco/src/domain/usecases/check_backend_readiness_usecase.dart';
import 'package:capibarco/src/domain/usecases/login_usecase.dart';
import 'package:capibarco/src/presentation/login_controller.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeAuthRepository implements AuthRepository {
  _FakeAuthRepository({required this.shouldLoginSucceed});

  final bool shouldLoginSucceed;

  @override
  Future<bool> checkBackendReadiness() async => true;

  @override
  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
  }) async {
    if (!shouldLoginSucceed) {
      throw Exception('login failed');
    }

    return Session(
      accessToken: 'token',
      refreshToken: 'refresh',
      userId: 'u1',
      email: email,
    );
  }
}

void main() {
  group('LoginController', () {
    test('validates email before login', () async {
      final repo = _FakeAuthRepository(shouldLoginSucceed: true);
      final controller = LoginController(
        loginUseCase: LoginUseCase(repo),
        checkBackendReadinessUseCase: CheckBackendReadinessUseCase(repo),
      );

      await controller.login(
        tenantId: 'default',
        email: 'invalid-email',
        password: 'password123',
      );

      expect(controller.error, 'Invalid email format');
      expect(controller.session, isNull);
    });

    test('stores session on successful login', () async {
      final repo = _FakeAuthRepository(shouldLoginSucceed: true);
      final controller = LoginController(
        loginUseCase: LoginUseCase(repo),
        checkBackendReadinessUseCase: CheckBackendReadinessUseCase(repo),
      );

      await controller.login(
        tenantId: 'default',
        email: 'user@example.com',
        password: 'password123',
      );

      expect(controller.session?.isAuthenticated, isTrue);
      expect(controller.error, isNull);
    });
  });
}
