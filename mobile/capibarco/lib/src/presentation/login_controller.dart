import 'package:flutter/foundation.dart';

import '../domain/entities/session.dart';
import '../domain/usecases/check_backend_readiness_usecase.dart';
import '../domain/usecases/login_usecase.dart';

class LoginController extends ChangeNotifier {
  LoginController({
    required LoginUseCase loginUseCase,
    required CheckBackendReadinessUseCase checkBackendReadinessUseCase,
  })  : _loginUseCase = loginUseCase,
        _checkBackendReadinessUseCase = checkBackendReadinessUseCase;

  final LoginUseCase _loginUseCase;
  final CheckBackendReadinessUseCase _checkBackendReadinessUseCase;

  Session? _session;
  bool _isLoading = false;
  String? _error;
  bool _backendReady = false;

  Session? get session => _session;
  bool get isLoading => _isLoading;
  String? get error => _error;
  bool get backendReady => _backendReady;

  Future<void> checkReadiness() async {
    _backendReady = await _checkBackendReadinessUseCase();
    notifyListeners();
  }

  Future<void> login({
    required String tenantId,
    required String email,
    required String password,
  }) async {
    final tenant = tenantId.trim();
    final normalizedEmail = email.trim();

    if (tenant.isEmpty) {
      _error = 'Tenant ID is required';
      notifyListeners();
      return;
    }

    if (!_isValidEmail(normalizedEmail)) {
      _error = 'Invalid email format';
      notifyListeners();
      return;
    }

    if (password.length < 8) {
      _error = 'Password must be at least 8 characters';
      notifyListeners();
      return;
    }

    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _session = await _loginUseCase(
        tenantId: tenant,
        email: normalizedEmail,
        password: password,
      );
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  bool _isValidEmail(String value) {
    final emailPattern = RegExp(r'^[^@\s]+@[^@\s]+\.[^@\s]+$');
    return emailPattern.hasMatch(value);
  }
}
