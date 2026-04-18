enum AppFlavor { local, staging, production }

enum TransportProtocol { rest, grpc, graphql }

enum ServiceKind { auth, feed, animals, profiles, notifications, chat, billing }

class ServiceConfig {
  const ServiceConfig({
    required this.baseUrl,
    required this.apiVersion,
    required this.protocol,
  });

  final String baseUrl;
  final String apiVersion;
  final TransportProtocol protocol;
}

class AppEnvironment {
  const AppEnvironment({
    required this.flavor,
    required this.gatewayBaseUrl,
    required this.apiVersion,
    required this.enableHttpLogs,
  });

  final AppFlavor flavor;
  final String gatewayBaseUrl;
  final String apiVersion;
  final bool enableHttpLogs;

  factory AppEnvironment.fromEnvironment() {
    const flavorName = String.fromEnvironment('APP_ENV', defaultValue: 'local');
    const gatewayBaseUrl = String.fromEnvironment(
      'API_GATEWAY_URL',
      defaultValue: 'http://localhost:18088',
    );
    const apiVersion = String.fromEnvironment(
      'API_VERSION',
      defaultValue: 'v1',
    );
    const enableHttpLogs = bool.fromEnvironment(
      'ENABLE_HTTP_LOGS',
      defaultValue: true,
    );

    return AppEnvironment(
      flavor: switch (flavorName) {
        'staging' => AppFlavor.staging,
        'production' => AppFlavor.production,
        _ => AppFlavor.local,
      },
      gatewayBaseUrl: gatewayBaseUrl,
      apiVersion: apiVersion,
      enableHttpLogs: enableHttpLogs,
    );
  }

  ServiceConfig service(ServiceKind kind) {
    return ServiceConfig(
      baseUrl: gatewayBaseUrl,
      apiVersion: apiVersion,
      protocol: TransportProtocol.rest,
    );
  }
}
