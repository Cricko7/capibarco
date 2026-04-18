import 'dart:convert';

import '../../../../core/storage/token_storage.dart';
import '../dtos/auth_models_dto.dart';

class AuthLocalDataSource {
  const AuthLocalDataSource(this._tokenStorage);

  static const _sessionKey = 'auth_session';

  final TokenStorage _tokenStorage;

  Future<void> clear() => _tokenStorage.delete(_sessionKey);

  Future<AuthSessionDto?> readSession() async {
    final raw = await _tokenStorage.read(_sessionKey);
    if (raw == null || raw.isEmpty) {
      return null;
    }
    return AuthSessionDto.fromJson(jsonDecode(raw) as Map<String, dynamic>);
  }

  Future<void> saveSession(AuthSessionDto session) async {
    await _tokenStorage.write(_sessionKey, jsonEncode(session.toJson()));
  }
}
