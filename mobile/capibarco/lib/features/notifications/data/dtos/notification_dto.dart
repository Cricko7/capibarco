import '../../domain/entities/notification_item.dart';

class NotificationItemDto {
  const NotificationItemDto({
    required this.id,
    required this.title,
    required this.body,
    required this.type,
    required this.status,
    required this.createdAt,
    required this.readAt,
  });

  final String id;
  final String title;
  final String body;
  final String type;
  final String status;
  final DateTime createdAt;
  final DateTime? readAt;

  factory NotificationItemDto.fromJson(Map<String, dynamic> json) {
    return NotificationItemDto(
      id: json['notification_id'] as String? ?? '',
      title: json['title'] as String? ?? '',
      body: json['body'] as String? ?? '',
      type: json['type'] as String? ?? 'NOTIFICATION_TYPE_UNSPECIFIED',
      status: json['status'] as String? ?? 'NOTIFICATION_STATUS_UNSPECIFIED',
      createdAt:
          DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
          DateTime.now().toUtc(),
      readAt: DateTime.tryParse(json['read_at'] as String? ?? ''),
    );
  }

  NotificationItemEntity toDomain() {
    return NotificationItemEntity(
      id: id,
      title: title,
      body: body,
      type: type.replaceAll('NOTIFICATION_TYPE_', '').toLowerCase(),
      status: status.replaceAll('NOTIFICATION_STATUS_', '').toLowerCase(),
      createdAt: createdAt,
      readAt: readAt,
    );
  }
}

class NotificationsPageDto {
  const NotificationsPageDto({
    required this.items,
    required this.nextPageToken,
    this.isStale = false,
  });

  final List<NotificationItemDto> items;
  final String nextPageToken;
  final bool isStale;

  factory NotificationsPageDto.fromJson(
    Map<String, dynamic> json, {
    bool isStale = false,
  }) {
    return NotificationsPageDto(
      items: (json['notifications'] as List<dynamic>? ?? const <dynamic>[])
          .map(
            (item) =>
                NotificationItemDto.fromJson(item as Map<String, dynamic>),
          )
          .toList(),
      nextPageToken:
          (json['page'] as Map<String, dynamic>? ??
                  const <String, dynamic>{})['next_page_token']
              as String? ??
          '',
      isStale: isStale,
    );
  }

  NotificationsPageEntity toDomain() {
    return NotificationsPageEntity(
      items: items.map((item) => item.toDomain()).toList(),
      nextPageToken: nextPageToken,
      isStale: isStale,
    );
  }
}
