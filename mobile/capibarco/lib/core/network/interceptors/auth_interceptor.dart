import 'package:dio/dio.dart';

import '../../../features/auth/domain/entities/auth_session.dart';

typedef AccessTokenReader = String? Function();
typedef SessionRefreshCallback = Future<AuthSession?> Function();
typedef SessionExpiredCallback = Future<void> Function();

class AuthInterceptor extends Interceptor {
  AuthInterceptor({
    required AccessTokenReader readAccessToken,
    required SessionRefreshCallback refreshSession,
    required SessionExpiredCallback onSessionExpired,
  }) : _readAccessToken = readAccessToken,
       _refreshSession = refreshSession,
       _onSessionExpired = onSessionExpired;

  final AccessTokenReader _readAccessToken;
  final SessionRefreshCallback _refreshSession;
  final SessionExpiredCallback _onSessionExpired;

  late Dio _dio;

  void attach(Dio dio) {
    _dio = dio;
  }

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    final requiresAuth = options.extra['authRequired'] != false;
    if (!requiresAuth) {
      handler.next(options);
      return;
    }

    final token = _readAccessToken();
    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    }
    handler.next(options);
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final statusCode = err.response?.statusCode;
    final alreadyRetried = err.requestOptions.extra['authRetried'] == true;
    final requiresAuth = err.requestOptions.extra['authRequired'] != false;

    if (statusCode != 401 || alreadyRetried || !requiresAuth) {
      handler.next(err);
      return;
    }

    final session = await _refreshSession();
    if (session == null) {
      await _onSessionExpired();
      handler.next(err);
      return;
    }

    final requestOptions = err.requestOptions;
    requestOptions.headers['Authorization'] = 'Bearer ${session.accessToken}';
    requestOptions.extra['authRetried'] = true;

    try {
      final response = await _dio.fetch<dynamic>(requestOptions);
      handler.resolve(response);
    } on DioException catch (refreshError) {
      await _onSessionExpired();
      handler.next(refreshError);
    }
  }
}
