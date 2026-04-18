import '../../../../core/network/rest_service_client.dart';
import '../dtos/notification_dto.dart';

class NotificationsApiClient {
  const NotificationsApiClient(this._client);

  final RestServiceClient _client;

  Future<NotificationsPageDto> listNotifications() async {
    final response = await _client.getMap(
      '/notifications',
      queryParameters: const <String, dynamic>{'page_size': 30},
    );
    return NotificationsPageDto.fromJson(response);
  }

  Future<Map<String, dynamic>> markAsRead(String notificationId) {
    return _client.postJson('/notifications/$notificationId/read');
  }
}
