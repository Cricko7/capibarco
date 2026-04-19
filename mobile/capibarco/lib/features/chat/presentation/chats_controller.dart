import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../../profile/data/api/profile_api_client.dart';
import '../domain/entities/chat_conversation.dart';
import 'chat_repository_provider.dart';

class ChatsState {
  const ChatsState({
    this.conversations = const <ChatConversationEntity>[],
    this.counterpartNames = const <String, String>{},
    this.isLoading = false,
    this.errorMessage,
  });

  final List<ChatConversationEntity> conversations;
  final Map<String, String> counterpartNames;
  final bool isLoading;
  final String? errorMessage;

  ChatsState copyWith({
    List<ChatConversationEntity>? conversations,
    Map<String, String>? counterpartNames,
    bool? isLoading,
    String? errorMessage,
    bool clearError = false,
  }) {
    return ChatsState(
      conversations: conversations ?? this.conversations,
      counterpartNames: counterpartNames ?? this.counterpartNames,
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
      final counterpartNames = await _loadCounterpartNames(conversations);
      state = state.copyWith(
        conversations: conversations,
        counterpartNames: counterpartNames,
        isLoading: false,
        clearError: true,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<Map<String, String>> _loadCounterpartNames(
    List<ChatConversationEntity> conversations,
  ) async {
    final currentProfileId =
        ref.read(authControllerProvider).session?.user.id ?? '';
    if (currentProfileId.isEmpty || conversations.isEmpty) {
      return const <String, String>{};
    }

    final counterpartIds = conversations
        .map((conversation) => conversation.counterpartProfileId(currentProfileId))
        .where((id) => id.isNotEmpty)
        .toSet()
        .toList();
    if (counterpartIds.isEmpty) {
      return const <String, String>{};
    }

    final environment = ref.read(appEnvironmentProvider);
    final profileApiClient = ProfileApiClient(
      RestServiceClient(
        dio: ref.read(authenticatedDioProvider),
        config: environment.service(ServiceKind.profiles),
      ),
    );

    final namesById = <String, String>{};
    await Future.wait<void>(
      counterpartIds.map((profileId) async {
        try {
          final profile = await profileApiClient.getProfile(profileId);
          final displayName = profile.displayName.trim();
          if (displayName.isNotEmpty) {
            namesById[profileId] = displayName;
          }
        } catch (_) {
          // Keep fallback in UI if profile name cannot be loaded.
        }
      }),
    );

    return namesById;
  }
}
