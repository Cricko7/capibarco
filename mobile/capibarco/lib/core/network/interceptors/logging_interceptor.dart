import 'package:dio/dio.dart';
import 'package:flutter/foundation.dart';

class AppLoggingInterceptor extends Interceptor {
  AppLoggingInterceptor({required this.enabled});

  final bool enabled;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler handler) {
    if (enabled && kDebugMode) {
      debugPrint(
        '[HTTP] ${options.method} ${options.uri} headers=${options.headers}',
      );
    }
    handler.next(options);
  }

  @override
  void onResponse(
    Response<dynamic> response,
    ResponseInterceptorHandler handler,
  ) {
    if (enabled && kDebugMode) {
      debugPrint(
        '[HTTP] ${response.statusCode} ${response.requestOptions.method} ${response.requestOptions.uri}',
      );
    }
    handler.next(response);
  }

  @override
  void onError(DioException err, ErrorInterceptorHandler handler) {
    if (enabled && kDebugMode) {
      debugPrint(
        '[HTTP] ERROR ${err.response?.statusCode} ${err.requestOptions.method} ${err.requestOptions.uri} ${err.message}',
      );
    }
    handler.next(err);
  }
}
