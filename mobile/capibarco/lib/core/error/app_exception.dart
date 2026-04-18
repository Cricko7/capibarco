class AppException implements Exception {
  const AppException({
    required this.message,
    this.statusCode,
    this.code,
    this.type,
    this.isRetryable = false,
  });

  final String message;
  final int? statusCode;
  final String? code;
  final String? type;
  final bool isRetryable;

  bool get isNotFound => statusCode == 404;

  bool get isUnauthorized => statusCode == 401;

  bool get isOfflineLike =>
      statusCode == null ||
      statusCode == 408 ||
      statusCode == 503 ||
      statusCode == 504;

  @override
  String toString() => message;
}
