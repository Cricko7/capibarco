import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../bootstrap/providers.dart';
import '../../../core/config/environment.dart';
import '../../../core/network/network_providers.dart';
import '../../../core/network/rest_service_client.dart';
import '../../auth/presentation/auth_controller.dart';
import '../data/api/feed_api_client.dart';
import '../data/datasources/feed_remote_data_source.dart';
import '../data/repositories/feed_repository_impl.dart';
import '../domain/entities/feed_models.dart';

class FeedState {
  const FeedState({
    this.cards = const <FeedCardEntity>[],
    this.nextPageToken = '',
    this.feedSessionId = '',
    this.isLoading = false,
    this.isLoadingMore = false,
    this.errorMessage,
    this.isStale = false,
  });

  final List<FeedCardEntity> cards;
  final String nextPageToken;
  final String feedSessionId;
  final bool isLoading;
  final bool isLoadingMore;
  final String? errorMessage;
  final bool isStale;

  FeedState copyWith({
    List<FeedCardEntity>? cards,
    String? nextPageToken,
    String? feedSessionId,
    bool? isLoading,
    bool? isLoadingMore,
    String? errorMessage,
    bool clearError = false,
    bool? isStale,
  }) {
    return FeedState(
      cards: cards ?? this.cards,
      nextPageToken: nextPageToken ?? this.nextPageToken,
      feedSessionId: feedSessionId ?? this.feedSessionId,
      isLoading: isLoading ?? this.isLoading,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      errorMessage: clearError ? null : (errorMessage ?? this.errorMessage),
      isStale: isStale ?? this.isStale,
    );
  }
}

final feedRepositoryProvider = Provider<FeedRepositoryImpl>((ref) {
  final environment = ref.watch(appEnvironmentProvider);
  return FeedRepositoryImpl(
    remoteDataSource: FeedRemoteDataSource(
      FeedApiClient(
        RestServiceClient(
          dio: ref.watch(authenticatedDioProvider),
          config: environment.service(ServiceKind.feed),
        ),
      ),
    ),
    cacheStore: ref.watch(cacheStoreProvider),
    errorMapper: ref.watch(errorMapperProvider),
  );
});

final feedControllerProvider = NotifierProvider<FeedController, FeedState>(
  FeedController.new,
);

class FeedController extends Notifier<FeedState> {
  FeedRepositoryImpl get _repository => ref.read(feedRepositoryProvider);

  @override
  FeedState build() => const FeedState();

  Future<void> load() async {
    state = state.copyWith(isLoading: true, clearError: true);
    try {
      final page = await _repository.getFeed();
      state = FeedState(
        cards: page.cards,
        nextPageToken: page.nextPageToken,
        feedSessionId: page.feedSessionId,
        isStale: page.isStale,
      );
    } catch (error) {
      state = state.copyWith(isLoading: false, errorMessage: error.toString());
    }
  }

  Future<void> loadMore() async {
    if (state.isLoadingMore || state.nextPageToken.isEmpty) {
      return;
    }

    state = state.copyWith(isLoadingMore: true, clearError: true);
    try {
      final page = await _repository.getFeed(pageToken: state.nextPageToken);
      state = state.copyWith(
        cards: <FeedCardEntity>[...state.cards, ...page.cards],
        nextPageToken: page.nextPageToken,
        feedSessionId: page.feedSessionId.isEmpty
            ? state.feedSessionId
            : page.feedSessionId,
        isLoadingMore: false,
        isStale: page.isStale,
      );
    } catch (error) {
      state = state.copyWith(
        isLoadingMore: false,
        errorMessage: error.toString(),
      );
    }
  }

  Future<void> swipe({
    required FeedCardEntity card,
    required bool liked,
  }) async {
    final profileId = ref
        .read(authControllerProvider.notifier)
        .currentProfileId;
    if (profileId == null) {
      return;
    }

    await _repository.swipeAnimal(
      animalId: card.animalId,
      ownerProfileId: profileId,
      liked: liked,
      feedCardId: card.id,
      feedSessionId: state.feedSessionId.isEmpty
          ? card.feedSessionId
          : state.feedSessionId,
    );

    final updated = state.cards.where((item) => item.id != card.id).toList();
    state = state.copyWith(cards: updated);
    if (updated.length < 3 && state.nextPageToken.isNotEmpty) {
      await loadMore();
    }
  }
}
