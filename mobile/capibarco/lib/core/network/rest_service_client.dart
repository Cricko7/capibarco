import 'package:dio/dio.dart';
import 'package:uuid/uuid.dart';

import '../config/environment.dart';

class RestServiceClient {
  RestServiceClient({required Dio dio, required ServiceConfig config})
    : _dio = dio,
      _config = config;

  final Dio _dio;
  final ServiceConfig _config;
  static const _uuid = Uuid();

  Future<Map<String, dynamic>> getMap(
    String path, {
    Map<String, dynamic>? queryParameters,
    bool requiresAuth = true,
    bool versioned = true,
  }) async {
    final response = await _dio.get<Map<String, dynamic>>(
      _composePath(path, versioned: versioned),
      queryParameters: queryParameters,
      options: Options(
        extra: <String, Object?>{
          'authRequired': requiresAuth,
          'retryable': true,
        },
      ),
    );
    return response.data ?? const <String, dynamic>{};
  }

  Future<Map<String, dynamic>> patchJson(
    String path, {
    required Map<String, dynamic> data,
    bool requiresAuth = true,
    bool versioned = true,
  }) async {
    final response = await _dio.patch<Map<String, dynamic>>(
      _composePath(path, versioned: versioned),
      data: data,
      options: Options(
        extra: <String, Object?>{
          'authRequired': requiresAuth,
          'retryable': false,
        },
      ),
    );
    return response.data ?? const <String, dynamic>{};
  }

  Future<Map<String, dynamic>> postJson(
    String path, {
    Map<String, dynamic>? data,
    bool requiresAuth = true,
    bool versioned = true,
    bool idempotent = false,
  }) async {
    final headers = <String, String>{};
    if (idempotent) {
      headers['Idempotency-Key'] = _uuid.v4();
    }

    final response = await _dio.post<Map<String, dynamic>>(
      _composePath(path, versioned: versioned),
      data: data,
      options: Options(
        headers: headers,
        extra: <String, Object?>{
          'authRequired': requiresAuth,
          'retryable': idempotent,
        },
      ),
    );
    return response.data ?? const <String, dynamic>{};
  }

  String _composePath(String path, {required bool versioned}) {
    if (!versioned) {
      return path;
    }

    final normalizedPath = path.startsWith('/') ? path : '/$path';
    final versionPrefix = '/${_config.apiVersion}';
    if (normalizedPath.startsWith(versionPrefix)) {
      return normalizedPath;
    }
    return '$versionPrefix$normalizedPath';
  }
}
