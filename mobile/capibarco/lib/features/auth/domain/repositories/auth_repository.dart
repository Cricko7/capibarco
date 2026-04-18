import '../entities/auth_session.dart';

abstract interface class AuthRepository {
  Future<AuthSession> login({
    required String email,
    required String password,
    required String locale,
  });

  Future<AuthSession> register({
    required String email,
    required String password,
    required String locale,
  });

  Future<AuthSession> refreshSession(String refreshToken);

  Future<AuthSession?> restoreSession();

  Future<void> clearSession();
}
