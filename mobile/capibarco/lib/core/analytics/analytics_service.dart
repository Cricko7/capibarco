abstract interface class AnalyticsService {
  Future<void> initialize();

  Future<void> trackEvent(
    String name, {
    Map<String, Object?> parameters = const <String, Object?>{},
  });

  Future<void> trackScreen(String name);
}

class NoopAnalyticsService implements AnalyticsService {
  const NoopAnalyticsService();

  @override
  Future<void> initialize() async {}

  @override
  Future<void> trackEvent(
    String name, {
    Map<String, Object?> parameters = const <String, Object?>{},
  }) async {}

  @override
  Future<void> trackScreen(String name) async {}
}
