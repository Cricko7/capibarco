class ChatMessageEntity {
  const ChatMessageEntity({
    required this.id,
    required this.senderProfileId,
    required this.text,
    required this.sentAt,
  });

  final String id;
  final String senderProfileId;
  final String text;
  final DateTime? sentAt;
}
