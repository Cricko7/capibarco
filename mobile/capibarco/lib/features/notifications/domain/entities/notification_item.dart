class NotificationItemEntity {
  const NotificationItemEntity({
    required this.id,
    required this.title,
    required this.body,
    required this.type,
    required this.status,
    required this.createdAt,
    this.readAt,
  });

  final String id;
  final String title;
  final String body;
  final String type;
  final String status;
  final DateTime createdAt;
  final DateTime? readAt;
}

class NotificationsPageEntity {
  const NotificationsPageEntity({
    required this.items,
    required this.nextPageToken,
    required this.isStale,
  });

  final List<NotificationItemEntity> items;
  final String nextPageToken;
  final bool isStale;
}
