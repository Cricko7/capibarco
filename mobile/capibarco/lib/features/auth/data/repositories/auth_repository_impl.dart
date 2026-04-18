import 'dart:convert';

import '../../domain/entities/auth_session.dart';
import '../../domain/repositories/auth_repository.dart';
import '../datasources/auth_local_data_source.dart';
import '../dtos/auth_models_dto.dart';

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl({
    required AuthLocalDataSource localDataSource,
  }) : _localDataSource = localDataSource;

  final AuthLocalDataSource _localDataSource;

  @override
  Future<void> clearSession() => _localDataSource.clear();

  @override
  Future<AuthSession> login({
    required String email,
    required String password,
    required String locale,
  }) async {
    // Temporary offline auth flow for mobile development without backend.
    final normalizedEmail = email.trim().toLowerCase();
    final safeEmail = normalizedEmail.isEmpty ? 'guest@local.dev' : normalizedEmail;
    final now = DateTime.now().toUtc();
    final session = AuthSessionDto(
      user: AuthUserDto(
        id: _buildLocalProfileId(safeEmail),
        tenantId: 'local',
        email: safeEmail,
        isActive: true,
      ),
      accessToken: _buildToken('access', safeEmail, now),
      refreshToken: _buildToken('refresh', safeEmail, now),
      expiresAt: now.add(const Duration(days: 30)),
    );
    await _localDataSource.saveSession(session);
    return session.toDomain();
  }

  @override
  Future<AuthSession> refreshSession(String refreshToken) async {
    final storedSession = await _localDataSource.readSession();
    final email = storedSession?.user.email ?? 'guest@local.dev';
    final now = DateTime.now().toUtc();
    final session = AuthSessionDto(
      user: storedSession?.user ??
          AuthUserDto(
            id: _buildLocalProfileId(email),
            tenantId: 'local',
            email: email,
            isActive: true,
          ),
      accessToken: _buildToken('access', email, now),
      refreshToken: storedSession?.refreshToken ?? _buildToken('refresh', email, now),
      expiresAt: now.add(const Duration(days: 30)),
    );
    await _localDataSource.saveSession(session);
    return session.toDomain();
  }

  @override
  Future<AuthSession> register({
    required String email,
    required String password,
    required String locale,
  }) => login(email: email, password: password, locale: locale);

  @override
  Future<AuthSession?> restoreSession() async {
    final session = await _localDataSource.readSession();
    return session?.toDomain();
  }

  String _buildLocalProfileId(String email) {
    return 'local-${email.hashCode.abs()}';
  }

  String _buildToken(String prefix, String email, DateTime now) {
    final payload = '$prefix:$email:${now.microsecondsSinceEpoch}';
    return base64Url.encode(utf8.encode(payload));
  }
}
