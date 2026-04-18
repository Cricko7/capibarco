import '../../domain/entities/chat_conversation.dart';

class ChatConversationDto {
  const ChatConversationDto({
    required this.id,
    required this.adopterProfileId,
    required this.ownerProfileId,
  });

  final String id;
  final String adopterProfileId;
  final String ownerProfileId;

  factory ChatConversationDto.fromJson(Map<String, dynamic> json) {
    final conversation = json['conversation'] as Map<String, dynamic>? ?? json;
    return ChatConversationDto(
      id: conversation['conversation_id'] as String? ?? '',
      adopterProfileId: conversation['adopter_profile_id'] as String? ?? '',
      ownerProfileId: conversation['owner_profile_id'] as String? ?? '',
    );
  }

  ChatConversationEntity toDomain() {
    return ChatConversationEntity(
      id: id,
      adopterProfileId: adopterProfileId,
      ownerProfileId: ownerProfileId,
    );
  }
}
