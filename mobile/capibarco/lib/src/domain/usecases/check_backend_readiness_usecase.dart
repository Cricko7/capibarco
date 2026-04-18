import '../repositories/auth_repository.dart';

class CheckBackendReadinessUseCase {
  const CheckBackendReadinessUseCase(this._repository);

  final AuthRepository _repository;

  Future<bool> call() => _repository.checkBackendReadiness();
}
