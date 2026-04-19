import '../../../../core/network/rest_service_client.dart';
import '../dtos/chat_conversation_dto.dart';
import '../dtos/chat_message_dto.dart';

class ChatApiClient {
  const ChatApiClient(this._client);

  final RestServiceClient _client;

  Future<ChatConversationDto> createConversation({
    required String targetProfileId,
    required String idempotencyKey,
    String animalId = '',
    String matchId = '',
  }) async {
    final response = await _client.postJson(
      '/chat/conversations',
      idempotencyKey: idempotencyKey,
      data: <String, dynamic>{
        'target_profile_id': targetProfileId,
        'animal_id': animalId,
        'match_id': matchId,
      },
    );
    return ChatConversationDto.fromJson(response);
  }

  Future<List<ChatConversationDto>> listConversations() async {
    final response = await _client.getMap(
      '/chat/conversations',
      queryParameters: const <String, dynamic>{'page_size': 50},
    );
    final items =
        response['conversations'] as List<dynamic>? ?? const <dynamic>[];
    return items
        .map(
          (item) => ChatConversationDto.fromJson(item as Map<String, dynamic>),
        )
        .toList();
  }

  Future<List<ChatMessageDto>> listMessages(String conversationId) async {
    final response = await _client.getMap(
      '/chat/conversations/$conversationId/messages',
      queryParameters: const <String, dynamic>{'page_size': 50},
    );
    final items = response['messages'] as List<dynamic>? ?? const <dynamic>[];
    return items
        .map((item) => ChatMessageDto.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<ChatMessageDto> sendMessage({
    required String conversationId,
    required String text,
    required String clientMessageId,
    required String idempotencyKey,
  }) async {
    final response = await _client.postJson(
      '/chat/conversations/$conversationId/messages',
      idempotencyKey: idempotencyKey,
      data: <String, dynamic>{
        'type': 1,
        'text': text,
        'client_message_id': clientMessageId,
      },
    );
    return ChatMessageDto.fromJson(response);
  }
}
