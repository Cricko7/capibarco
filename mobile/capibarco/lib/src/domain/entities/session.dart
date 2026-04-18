class Session {
  const Session({
    required this.accessToken,
    required this.refreshToken,
    required this.userId,
    required this.email,
  });

  final String accessToken;
  final String refreshToken;
  final String userId;
  final String email;

  bool get isAuthenticated => accessToken.isNotEmpty && userId.isNotEmpty;
}
