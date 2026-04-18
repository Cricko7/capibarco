import '../entities/session.dart';
import '../repositories/auth_repository.dart';

class LoginUseCase {
  const LoginUseCase(this._repository);

  final AuthRepository _repository;

  Future<Session> call({
    required String tenantId,
    required String email,
    required String password,
  }) {
    return _repository.login(
      tenantId: tenantId,
      email: email,
      password: password,
    );
  }
}
