class ChatConversationEntity {
  const ChatConversationEntity({
    required this.id,
    required this.adopterProfileId,
    required this.ownerProfileId,
  });

  final String id;
  final String adopterProfileId;
  final String ownerProfileId;

  String counterpartProfileId(String currentProfileId) {
    if (currentProfileId == adopterProfileId) {
      return ownerProfileId;
    }
    if (currentProfileId == ownerProfileId) {
      return adopterProfileId;
    }
    return adopterProfileId.isNotEmpty ? adopterProfileId : ownerProfileId;
  }
}
