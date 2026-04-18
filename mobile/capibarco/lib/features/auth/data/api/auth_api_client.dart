import '../../../../core/network/rest_service_client.dart';
import '../dtos/auth_models_dto.dart';

class AuthApiClient {
  const AuthApiClient(this._client);

  final RestServiceClient _client;

  Future<AuthSessionDto> login(LoginRequestDto request) async {
    final response = await _client.postJson(
      '/auth/login',
      requiresAuth: false,
      data: request.toJson(),
    );
    return AuthSessionDto.fromJson(response);
  }

  Future<AuthSessionDto> refresh(RefreshTokenRequestDto request) async {
    final response = await _client.postJson(
      '/auth/refresh',
      requiresAuth: false,
      data: request.toJson(),
    );
    return AuthSessionDto.fromJson(response);
  }

  Future<AuthSessionDto> register(RegisterRequestDto request) async {
    final response = await _client.postJson(
      '/auth/register',
      requiresAuth: false,
      data: request.toJson(),
    );
    return AuthSessionDto.fromJson(response);
  }
}
