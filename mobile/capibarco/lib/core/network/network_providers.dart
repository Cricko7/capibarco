import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../bootstrap/providers.dart';
import '../../features/auth/presentation/auth_controller.dart';
import 'interceptors/auth_interceptor.dart';
import 'interceptors/logging_interceptor.dart';
import 'interceptors/retry_interceptor.dart';

final authenticatedDioProvider = Provider<Dio>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  final authController = ref.read(authControllerProvider.notifier);
  final dio = Dio(
    BaseOptions(
      baseUrl: environment.gatewayBaseUrl,
      connectTimeout: const Duration(seconds: 12),
      receiveTimeout: const Duration(seconds: 12),
      sendTimeout: const Duration(seconds: 12),
      contentType: Headers.jsonContentType,
      responseType: ResponseType.json,
    ),
  );

  final authInterceptor = AuthInterceptor(
    readAccessToken: () => authController.accessToken,
    refreshSession: authController.refreshSession,
    onSessionExpired: authController.handleSessionExpired,
  )..attach(dio);
  final retryInterceptor = RetryInterceptor()..attach(dio);

  dio.interceptors.addAll(<Interceptor>[
    authInterceptor,
    retryInterceptor,
    AppLoggingInterceptor(enabled: environment.enableHttpLogs),
  ]);

  return dio;
});
