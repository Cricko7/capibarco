import '../api/notifications_api_client.dart';
import '../dtos/notification_dto.dart';

class NotificationsRemoteDataSource {
  const NotificationsRemoteDataSource(this._apiClient);

  final NotificationsApiClient _apiClient;

  Future<NotificationsPageDto> listNotifications() =>
      _apiClient.listNotifications();

  Future<Map<String, dynamic>> markAsRead(String notificationId) =>
      _apiClient.markAsRead(notificationId);
}
