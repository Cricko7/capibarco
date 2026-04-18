import 'package:flutter/material.dart';

import 'src/data/datasources/auth_api_client.dart';
import 'src/data/repositories/auth_repository_impl.dart';
import 'src/domain/usecases/check_backend_readiness_usecase.dart';
import 'src/domain/usecases/login_usecase.dart';
import 'src/presentation/login_controller.dart';
import 'src/presentation/login_page.dart';

void main() {
  const baseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://10.0.2.2:8080',
  );
  const loginPath = String.fromEnvironment(
    'API_LOGIN_PATH',
    defaultValue: '/v1/login',
  );

  final apiClient = AuthApiClient(
    baseUrl: baseUrl,
    loginPath: loginPath,
  );
  final repository = AuthRepositoryImpl(apiClient);
  final loginUseCase = LoginUseCase(repository);
  final readinessUseCase = CheckBackendReadinessUseCase(repository);
  final controller = LoginController(
    loginUseCase: loginUseCase,
    checkBackendReadinessUseCase: readinessUseCase,
  );

  runApp(CapibarcoApp(controller: controller));
}

class CapibarcoApp extends StatelessWidget {
  const CapibarcoApp({required this.controller, super.key});

  final LoginController controller;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Capibarco',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.green),
        useMaterial3: true,
      ),
      home: LoginPage(controller: controller),
    );
  }
}
