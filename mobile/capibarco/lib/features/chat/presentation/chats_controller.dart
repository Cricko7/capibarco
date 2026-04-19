import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/entities/chat_conversation.dart';
import 'chat_repository_provider.dart';

class ChatsState {
  const ChatsState({
    this.conversations = const <ChatConversationEntity>[],
    this.isLoading = false,
    this.errorMessage,
  });

  final List<ChatConversationEntity> conversations;
  final bool isLoading;
  final String? errorMessage;

  ChatsState copyWith({
    List<ChatConversationEntity>? conversations,
    bool? isLoading,
    String? errorMessage,
    bool clearError = false,
  }) {
    return ChatsState(
      conversations: conversations ?? this.conversations,
      isLoading: isLoading ?? this.isLoading,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
    );
  }
}

final chatsControllerProvider = NotifierProvider<ChatsController, ChatsState>(
  ChatsController.new,
);

class ChatsController extends Notifier<ChatsState> {
  @override
  ChatsState build() => const ChatsState();

  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final conversations = await ref
          .read(chatRepositoryProvider)
          .listConversations();
      state = state.copyWith(
        conversations: conversations,
        isLoading: false,
        clearError: true,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }
}
