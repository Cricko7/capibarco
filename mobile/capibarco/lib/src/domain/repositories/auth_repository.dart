import '../entities/session.dart';

abstract class AuthRepository {
  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
  });

  Future<bool> checkBackendReadiness();
}
