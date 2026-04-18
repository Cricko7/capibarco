import '../../domain/entities/chat_message.dart';

class ChatMessageDto {
  const ChatMessageDto({
    required this.id,
    required this.senderProfileId,
    required this.text,
    required this.sentAt,
  });

  final String id;
  final String senderProfileId;
  final String text;
  final DateTime? sentAt;

  factory ChatMessageDto.fromJson(Map<String, dynamic> json) {
    final message = json['message'] as Map<String, dynamic>? ?? json;
    return ChatMessageDto(
      id: message['message_id'] as String? ?? '',
      senderProfileId: message['sender_profile_id'] as String? ?? '',
      text: message['text'] as String? ?? '',
      sentAt: DateTime.tryParse(message['sent_at'] as String? ?? ''),
    );
  }

  ChatMessageEntity toDomain() {
    return ChatMessageEntity(
      id: id,
      senderProfileId: senderProfileId,
      text: text,
      sentAt: sentAt,
    );
  }
}
