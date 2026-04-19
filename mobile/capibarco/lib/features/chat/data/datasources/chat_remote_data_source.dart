import '../api/chat_api_client.dart';
import '../dtos/chat_conversation_dto.dart';
import '../dtos/chat_message_dto.dart';

class ChatRemoteDataSource {
  const ChatRemoteDataSource(this._apiClient);

  final ChatApiClient _apiClient;

  Future<ChatConversationDto> createConversation({
    required String targetProfileId,
    required String idempotencyKey,
    String animalId = '',
    String matchId = '',
  }) => _apiClient.createConversation(
    targetProfileId: targetProfileId,
    idempotencyKey: idempotencyKey,
    animalId: animalId,
    matchId: matchId,
  );

  Future<List<ChatConversationDto>> listConversations() =>
      _apiClient.listConversations();

  Future<List<ChatMessageDto>> listMessages(String conversationId) =>
      _apiClient.listMessages(conversationId);

  Future<ChatMessageDto> sendMessage({
    required String conversationId,
    required String text,
    required String clientMessageId,
    required String idempotencyKey,
  }) => _apiClient.sendMessage(
    conversationId: conversationId,
    text: text,
    clientMessageId: clientMessageId,
    idempotencyKey: idempotencyKey,
  );
}
