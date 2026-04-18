import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

class AuthApiClient {
  AuthApiClient({
    required this.baseUrl,
    this.loginPath = '/v1/login',
    this.readinessPath = '/readyz',
    Duration timeout = const Duration(seconds: 10),
    http.Client? client,
  })  : _timeout = timeout,
        _client = client ?? http.Client();

  final String baseUrl;
  final String loginPath;
  final String readinessPath;
  final Duration _timeout;
  final http.Client _client;

  Future<Map<String, dynamic>> login({
    required String tenantId,
    required String email,
    required String password,
  }) async {
    final uri = Uri.parse('$baseUrl$loginPath');
    final response = await _client
        .post(
          uri,
          headers: const {'Content-Type': 'application/json'},
          body: jsonEncode(
            {
              'tenant_id': tenantId,
              'email': email,
              'password': password,
            },
          ),
        )
        .timeout(_timeout);

    if (response.statusCode != 200) {
      throw AuthApiException(
        'Login failed: HTTP ${response.statusCode}',
      );
    }

    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic>) {
      throw const AuthApiException('Login failed: unexpected response format');
    }

    return decoded;
  }

  Future<bool> checkReadiness() async {
    try {
      final uri = Uri.parse('$baseUrl$readinessPath');
      final response = await _client.get(uri).timeout(_timeout);
      return response.statusCode == 200;
    } on TimeoutException {
      return false;
    } on http.ClientException {
      return false;
    }
  }
}

class AuthApiException implements Exception {
  const AuthApiException(this.message);

  final String message;

  @override
  String toString() => message;
}
