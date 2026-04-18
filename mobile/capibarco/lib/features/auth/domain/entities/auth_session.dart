class AuthUser {
  const AuthUser({
    required this.id,
    required this.tenantId,
    required this.email,
    required this.isActive,
  });

  final String id;
  final String tenantId;
  final String email;
  final bool isActive;
}

class AuthSession {
  const AuthSession({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
    required this.expiresAt,
  });

  final AuthUser user;
  final String accessToken;
  final String refreshToken;
  final DateTime expiresAt;

  bool isExpiringWithin(Duration threshold) {
    return DateTime.now().toUtc().isAfter(
      expiresAt.toUtc().subtract(threshold),
    );
  }
}
