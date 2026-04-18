import '../api/auth_api_client.dart';
import '../dtos/auth_models_dto.dart';

class AuthRemoteDataSource {
  const AuthRemoteDataSource(this._apiClient);

  final AuthApiClient _apiClient;

  Future<AuthSessionDto> login(LoginRequestDto request) =>
      _apiClient.login(request);

  Future<AuthSessionDto> refresh(RefreshTokenRequestDto request) =>
      _apiClient.refresh(request);

  Future<AuthSessionDto> register(RegisterRequestDto request) =>
      _apiClient.register(request);
}
