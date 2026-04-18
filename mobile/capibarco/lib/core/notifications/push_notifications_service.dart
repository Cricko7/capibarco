abstract interface class PushNotificationsService {
  Future<void> initialize();

  Future<void> registerDeviceToken(String token);
}

class NoopPushNotificationsService implements PushNotificationsService {
  const NoopPushNotificationsService();

  @override
  Future<void> initialize() async {}

  @override
  Future<void> registerDeviceToken(String token) async {}
}
