import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/error/error_mapper.dart';
import '../data/datasources/auth_local_data_source.dart';
import '../data/repositories/auth_repository_impl.dart';
import '../domain/entities/auth_session.dart';
import '../domain/repositories/auth_repository.dart';
import 'auth_state.dart';

final errorMapperProvider = Provider<ErrorMapper>((_) => const ErrorMapper());

final authRepositoryProvider = Provider<AuthRepository>((ref) {
  return AuthRepositoryImpl(
    localDataSource: AuthLocalDataSource(ref.watch(tokenStorageProvider)),
  );
});

final authControllerProvider = NotifierProvider<AuthController, AuthState>(
  AuthController.new,
);

class AuthController extends Notifier<AuthState> {
  Future<AuthSession?>? _refreshInFlight;

  AuthRepository get _repository => ref.read(authRepositoryProvider);
  ErrorMapper get _errorMapper => ref.read(errorMapperProvider);

  @override
  AuthState build() => const AuthState.initial();

  String? get accessToken => state.session?.accessToken;

  String? get currentProfileId => state.session?.user.id;

  Future<void> bootstrap() async {
    if (!state.isBootstrapping && state.status != AuthStatus.initial) {
      return;
    }

    state = state.copyWith(isBootstrapping: true, clearError: true);

    final session = await _repository.restoreSession();
    if (session == null) {
      state = const AuthState(
        status: AuthStatus.unauthenticated,
        isBootstrapping: false,
      );
      return;
    }

    state = AuthState(
      status: AuthStatus.authenticated,
      session: session,
      isBootstrapping: false,
    );

    if (session.isExpiringWithin(const Duration(minutes: 1))) {
      await refreshSession();
    }
  }

  Future<void> login({
    required String email,
    required String password,
    required String locale,
  }) async {
    state = state.copyWith(isSubmitting: true, clearError: true);
    try {
      final session = await _repository.login(
        email: email,
        password: password,
        locale: locale,
      );
      state = AuthState(
        status: AuthStatus.authenticated,
        session: session,
        isSubmitting: false,
        isBootstrapping: false,
      );
    } catch (error) {
      state = state.copyWith(
        status: AuthStatus.unauthenticated,
        errorMessage: _errorMapper.map(error).message,
        isSubmitting: false,
        isBootstrapping: false,
        clearSession: true,
      );
    }
  }

  Future<void> register({
    required String email,
    required String password,
    required String locale,
  }) async {
    state = state.copyWith(isSubmitting: true, clearError: true);
    try {
      final session = await _repository.register(
        email: email,
        password: password,
        locale: locale,
      );
      state = AuthState(
        status: AuthStatus.authenticated,
        session: session,
        isSubmitting: false,
        isBootstrapping: false,
      );
    } catch (error) {
      state = state.copyWith(
        status: AuthStatus.unauthenticated,
        errorMessage: _errorMapper.map(error).message,
        isSubmitting: false,
        isBootstrapping: false,
        clearSession: true,
      );
    }
  }

  Future<void> logout() async {
    await _repository.clearSession();
    state = const AuthState(
      status: AuthStatus.unauthenticated,
      isBootstrapping: false,
    );
  }

  Future<AuthSession?> refreshSession() async {
    if (_refreshInFlight != null) {
      return _refreshInFlight!;
    }

    final refreshToken = state.session?.refreshToken;
    if (refreshToken == null || refreshToken.isEmpty) {
      await logout();
      return null;
    }

    final completer = _refresh(refreshToken);
    _refreshInFlight = completer;

    try {
      return await completer;
    } finally {
      _refreshInFlight = null;
    }
  }

  Future<void> handleSessionExpired() => logout();

  Future<AuthSession?> _refresh(String refreshToken) async {
    try {
      final session = await _repository.refreshSession(refreshToken);
      state = AuthState(
        status: AuthStatus.authenticated,
        session: session,
        isBootstrapping: false,
      );
      return session;
    } catch (_) {
      await logout();
      return null;
    }
  }
}
