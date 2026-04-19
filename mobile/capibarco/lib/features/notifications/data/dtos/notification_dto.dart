import '../../domain/entities/notification_item.dart';

class NotificationItemDto {
  const NotificationItemDto({
    required this.id,
    required this.title,
    required this.body,
    required this.type,
    required this.status,
    required this.createdAt,
    required this.data,
    required this.readAt,
  });

  final String id;
  final String title;
  final String body;
  final String type;
  final String status;
  final DateTime createdAt;
  final Map<String, String> data;
  final DateTime? readAt;

  factory NotificationItemDto.fromJson(Map<String, dynamic> json) {
    return NotificationItemDto(
      id: _stringValue(_field(json, 'notification_id', 'notificationId')),
      title: _stringValue(json['title']),
      body: _stringValue(json['body']),
      type: _notificationTypeValue(json['type']),
      status: _notificationStatusValue(json['status']),
      data: _stringMap(json['data']),
      createdAt:
          DateTime.tryParse(
            _stringValue(_field(json, 'created_at', 'createdAt')),
          )?.toUtc() ??
          DateTime.now().toUtc(),
      readAt: DateTime.tryParse(
        _stringValue(_field(json, 'read_at', 'readAt')),
      ),
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
      data: data,
      readAt: readAt,
    );
  }
}

Map<String, String> _stringMap(Object? value) {
  if (value is! Map) {
    return const <String, String>{};
  }
  return value.map(
    (key, mapValue) => MapEntry(key.toString(), _stringValue(mapValue)),
  );
}

String _stringValue(Object? value, {String fallback = ''}) {
  if (value == null) {
    return fallback;
  }
  if (value is String) {
    return value;
  }
  return value.toString();
}

Object? _field(Map<String, dynamic> json, String snakeName, String camelName) {
  if (json.containsKey(snakeName)) {
    return json[snakeName];
  }
  return json[camelName];
}

String _notificationTypeValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'NOTIFICATION_TYPE_MATCH_CREATED',
      2 => 'NOTIFICATION_TYPE_CHAT_MESSAGE',
      3 => 'NOTIFICATION_TYPE_DONATION_SUCCEEDED',
      4 => 'NOTIFICATION_TYPE_BOOST_ACTIVATED',
      5 => 'NOTIFICATION_TYPE_REVIEW_CREATED',
      _ => 'NOTIFICATION_TYPE_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'NOTIFICATION_TYPE_UNSPECIFIED');
}

String _notificationStatusValue(Object? value) {
  if (value is num) {
    return switch (value.toInt()) {
      1 => 'NOTIFICATION_STATUS_PENDING',
      2 => 'NOTIFICATION_STATUS_DELIVERED',
      3 => 'NOTIFICATION_STATUS_FAILED',
      4 => 'NOTIFICATION_STATUS_READ',
      _ => 'NOTIFICATION_STATUS_UNSPECIFIED',
    };
  }
  return _stringValue(value, fallback: 'NOTIFICATION_STATUS_UNSPECIFIED');
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
          ((json['page'] as Map<String, dynamic>? ??
                  const <String, dynamic>{})['nextPageToken'] as String?) ??
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
