import 'dart:async';

import 'package:dio/dio.dart';

class RetryInterceptor extends Interceptor {
  RetryInterceptor({this.maxRetries = 2});

  final int maxRetries;

  late Dio _dio;

  void attach(Dio dio) {
    _dio = dio;
  }

  @override
  Future<void> onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    final options = err.requestOptions;
    final currentRetry = options.extra['retryCount'] as int? ?? 0;

    if (!_shouldRetry(err, options) || currentRetry >= maxRetries) {
      handler.next(err);
      return;
    }

    options.extra['retryCount'] = currentRetry + 1;
    await Future<void>.delayed(
      Duration(milliseconds: 300 * (currentRetry + 1)),
    );

    try {
      final response = await _dio.fetch<dynamic>(options);
      handler.resolve(response);
    } on DioException catch (retryError) {
      handler.next(retryError);
    }
  }

  bool _shouldRetry(DioException err, RequestOptions options) {
    final statusCode = err.response?.statusCode;
    final method = options.method.toUpperCase();
    final hasIdempotencyKey =
        options.headers.containsKey('Idempotency-Key') ||
        options.headers.containsKey('X-Idempotency-Key');
    final safeMethod =
        method == 'GET' || method == 'HEAD' || method == 'OPTIONS';
    final retryableMethod =
        safeMethod || hasIdempotencyKey || options.extra['retryable'] == true;

    if (!retryableMethod) {
      return false;
    }

    return err.type == DioExceptionType.connectionTimeout ||
        err.type == DioExceptionType.receiveTimeout ||
        err.type == DioExceptionType.sendTimeout ||
        err.type == DioExceptionType.connectionError ||
        statusCode == 429 ||
        statusCode == 503 ||
        statusCode == 504;
  }
}
