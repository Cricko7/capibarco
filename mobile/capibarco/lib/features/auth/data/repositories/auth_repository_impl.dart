import '../../domain/entities/auth_session.dart';
import '../../domain/repositories/auth_repository.dart';
import '../datasources/auth_local_data_source.dart';
import '../datasources/auth_remote_data_source.dart';
import '../dtos/auth_models_dto.dart';

class AuthRepositoryImpl implements AuthRepository {
  const AuthRepositoryImpl({
    required AuthRemoteDataSource remoteDataSource,
    required AuthLocalDataSource localDataSource,
  }) : _remoteDataSource = remoteDataSource,
       _localDataSource = localDataSource;

  final AuthRemoteDataSource _remoteDataSource;
  final AuthLocalDataSource _localDataSource;

  @override
  Future<void> clearSession() => _localDataSource.clear();

  @override
  Future<AuthSession> login({
    required String email,
    required String password,
    required String locale,
  }) async {
    final session = await _remoteDataSource.login(
      LoginRequestDto(email: email, password: password, locale: locale),
    );
    await _localDataSource.saveSession(session);
    return session.toDomain();
  }

  @override
  Future<AuthSession> refreshSession(String refreshToken) async {
    final session = await _remoteDataSource.refresh(
      RefreshTokenRequestDto(refreshToken: refreshToken),
    );
    await _localDataSource.saveSession(session);
    return session.toDomain();
  }

  @override
  Future<AuthSession> register({
    required String email,
    required String password,
    required String locale,
  }) async {
    final session = await _remoteDataSource.register(
      RegisterRequestDto(email: email, password: password, locale: locale),
    );
    await _localDataSource.saveSession(session);
    return session.toDomain();
  }

  @override
  Future<AuthSession?> restoreSession() async {
    final session = await _localDataSource.readSession();
    return session?.toDomain();
  }
}
