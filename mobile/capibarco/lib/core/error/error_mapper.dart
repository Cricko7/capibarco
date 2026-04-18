import 'dart:io';

import 'package:dio/dio.dart';

import 'app_exception.dart';

class ErrorMapper {
  const ErrorMapper();

  AppException map(Object error) {
    if (error is AppException) {
      return error;
    }

    if (error is DioException) {
      final responseData = error.response?.data;
      final payload = responseData is Map<String, dynamic>
          ? responseData
          : const <String, dynamic>{};
      final message =
          payload['detail'] as String? ??
          payload['message'] as String? ??
          error.message ??
          'Unexpected network error';
      final statusCode = error.response?.statusCode;

      return AppException(
        message: message,
        statusCode: statusCode,
        code: payload['code'] as String?,
        type: payload['type'] as String?,
        isRetryable: _isRetryable(error),
      );
    }

    if (error is SocketException) {
      return const AppException(
        message: 'No internet connection',
        isRetryable: true,
      );
    }

    return AppException(message: error.toString());
  }

  bool _isRetryable(DioException error) {
    return error.type == DioExceptionType.connectionTimeout ||
        error.type == DioExceptionType.receiveTimeout ||
        error.type == DioExceptionType.sendTimeout ||
        error.type == DioExceptionType.connectionError ||
        error.response?.statusCode == 429 ||
        error.response?.statusCode == 503 ||
        error.response?.statusCode == 504;
  }
}
