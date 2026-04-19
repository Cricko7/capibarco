import 'package:uuid/uuid.dart';

import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/chat_conversation.dart';
import '../../domain/entities/chat_message.dart';
import '../datasources/chat_remote_data_source.dart';

class ChatRepositoryImpl {
  ChatRepositoryImpl({
    required ChatRemoteDataSource remoteDataSource,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _errorMapper = errorMapper;

  final ChatRemoteDataSource _remoteDataSource;
  final ErrorMapper _errorMapper;
  static const _uuid = Uuid();

  Future<ChatConversationEntity> createConversation({
    required String targetProfileId,
    required String idempotencyKey,
    String animalId = '',
    String matchId = '',
  }) async {
    try {
      try {
        final existingConversation = await _findConversationByTarget(
          targetProfileId,
        );
        if (existingConversation != null) {
          return existingConversation;
        }
      } catch (_) {
        // If lookup fails we still try to create a conversation.
      }

      final conversation = await _remoteDataSource.createConversation(
        targetProfileId: targetProfileId,
        idempotencyKey: idempotencyKey,
        animalId: animalId,
        matchId: matchId,
      );
      return conversation.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<List<ChatMessageEntity>> listMessages(String conversationId) async {
    try {
      final messages = await _remoteDataSource.listMessages(conversationId);
      return messages.map((item) => item.toDomain()).toList();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<List<ChatConversationEntity>> listConversations() async {
    try {
      final conversations = await _remoteDataSource.listConversations();
      final unique = <String, ChatConversationEntity>{};
      for (final conversation in conversations) {
        final ids = <String>[
          conversation.adopterProfileId,
          conversation.ownerProfileId,
        ]..sort();
        final pairKey = ids.join(':');
        unique.putIfAbsent(pairKey, () => conversation.toDomain());
      }
      return unique.values.toList();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }

  Future<ChatConversationEntity?> _findConversationByTarget(
    String targetProfileId,
  ) async {
    final conversations = await _remoteDataSource.listConversations();
    final existing = conversations.where((conversation) {
      return conversation.adopterProfileId == targetProfileId ||
          conversation.ownerProfileId == targetProfileId;
    });
    if (existing.isEmpty) {
      return null;
    }
    return existing.first.toDomain();
  }

  Future<ChatMessageEntity> sendMessage({
    required String conversationId,
    required String text,
  }) async {
    try {
      final message = await _remoteDataSource.sendMessage(
        conversationId: conversationId,
        text: text,
        clientMessageId: _uuid.v4(),
        idempotencyKey: _uuid.v4(),
      );
      return message.toDomain();
    } catch (error) {
      throw _errorMapper.map(error);
    }
  }
}
