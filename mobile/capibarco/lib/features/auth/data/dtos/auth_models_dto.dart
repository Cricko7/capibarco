import '../../domain/entities/auth_session.dart';

class AuthUserDto {
  const AuthUserDto({
    required this.id,
    required this.tenantId,
    required this.email,
    required this.isActive,
  });

  final String id;
  final String tenantId;
  final String email;
  final bool isActive;

  factory AuthUserDto.fromJson(Map<String, dynamic> json) {
    return AuthUserDto(
      id: json['id'] as String? ?? '',
      tenantId: json['tenant_id'] as String? ?? 'default',
      email: json['email'] as String? ?? '',
      isActive: json['is_active'] as bool? ?? true,
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'tenant_id': tenantId,
      'email': email,
      'is_active': isActive,
    };
  }

  AuthUser toDomain() {
    return AuthUser(
      id: id,
      tenantId: tenantId,
      email: email,
      isActive: isActive,
    );
  }
}

class AuthSessionDto {
  const AuthSessionDto({
    required this.user,
    required this.accessToken,
    required this.refreshToken,
    required this.expiresAt,
  });

  final AuthUserDto user;
  final String accessToken;
  final String refreshToken;
  final DateTime expiresAt;

  factory AuthSessionDto.fromJson(Map<String, dynamic> json) {
    return AuthSessionDto(
      user: AuthUserDto.fromJson(
        json['user'] as Map<String, dynamic>? ?? const <String, dynamic>{},
      ),
      accessToken: json['access_token'] as String? ?? '',
      refreshToken: json['refresh_token'] as String? ?? '',
      expiresAt:
          DateTime.tryParse(json['expires_at'] as String? ?? '')?.toUtc() ??
          DateTime.now().toUtc(),
    );
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'user': user.toJson(),
      'access_token': accessToken,
      'refresh_token': refreshToken,
      'expires_at': expiresAt.toIso8601String(),
    };
  }

  AuthSession toDomain() {
    return AuthSession(
      user: user.toDomain(),
      accessToken: accessToken,
      refreshToken: refreshToken,
      expiresAt: expiresAt,
    );
  }
}

class LoginRequestDto {
  const LoginRequestDto({
    required this.email,
    required this.password,
    required this.locale,
  });

  final String email;
  final String password;
  final String locale;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'email': email,
      'password': password,
      'locale': locale,
    };
  }
}

class RegisterRequestDto {
  const RegisterRequestDto({
    required this.email,
    required this.password,
    required this.locale,
  });

  final String email;
  final String password;
  final String locale;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'email': email,
      'password': password,
      'locale': locale,
    };
  }
}

class RefreshTokenRequestDto {
  const RefreshTokenRequestDto({required this.refreshToken});

  final String refreshToken;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{'refresh_token': refreshToken};
  }
}
