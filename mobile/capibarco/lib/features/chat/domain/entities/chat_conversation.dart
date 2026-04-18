class ChatConversationEntity {
  const ChatConversationEntity({
    required this.id,
    required this.adopterProfileId,
    required this.ownerProfileId,
  });

  final String id;
  final String adopterProfileId;
  final String ownerProfileId;
}
