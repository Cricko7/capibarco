import '../domain/entities/auth_session.dart';

enum AuthStatus { initial, authenticated, unauthenticated }

class AuthState {
  const AuthState({
    required this.status,
    this.session,
    this.errorMessage,
    this.isSubmitting = false,
    this.isBootstrapping = false,
  });

  const AuthState.initial()
    : status = AuthStatus.initial,
      session = null,
      errorMessage = null,
      isSubmitting = false,
      isBootstrapping = true;

  final AuthStatus status;
  final AuthSession? session;
  final String? errorMessage;
  final bool isSubmitting;
  final bool isBootstrapping;

  bool get isAuthenticated =>
      status == AuthStatus.authenticated && session != null;

  AuthState copyWith({
    AuthStatus? status,
    AuthSession? session,
    bool clearSession = false,
    String? errorMessage,
    bool clearError = false,
    bool? isSubmitting,
    bool? isBootstrapping,
  }) {
    return AuthState(
      status: status ?? this.status,
      session: clearSession ? null : (session ?? this.session),
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      isSubmitting: isSubmitting ?? this.isSubmitting,
      isBootstrapping: isBootstrapping ?? this.isBootstrapping,
    );
  }
}
