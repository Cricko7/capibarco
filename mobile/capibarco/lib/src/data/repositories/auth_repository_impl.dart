import '../../domain/entities/session.dart';
import '../../domain/repositories/auth_repository.dart';
import '../datasources/auth_api_client.dart';

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl(this._apiClient);

  final AuthApiClient _apiClient;

  @override
  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
  }) async {
    final payload = await _apiClient.login(
      tenantId: tenantId,
      email: email,
      password: password,
    );

    final accessToken = _readString(payload, ['access_token', 'accessToken']);
    final refreshToken = _readString(payload, ['refresh_token', 'refreshToken']);
    final user = payload['user'];
    final userMap = user is Map<String, dynamic> ? user : const <String, dynamic>{};
    final userId = _readString(userMap, ['id', 'user_id', 'userId']);
    final resolvedEmail = _readString(userMap, ['email'], fallback: email);

    if (accessToken.isEmpty || userId.isEmpty) {
      throw const AuthApiException('Login failed: missing mandatory fields in response');
    }

    return Session(
      accessToken: accessToken,
      refreshToken: refreshToken,
      userId: userId,
      email: resolvedEmail,
    );
  }

  @override
  Future<bool> checkBackendReadiness() => _apiClient.checkReadiness();

  String _readString(Map<String, dynamic> source, List<String> keys, {String fallback = ''}) {
    for (final key in keys) {
      final value = source[key];
      if (value is String && value.isNotEmpty) {
        return value;
      }
    }
    return fallback;
  }
}
