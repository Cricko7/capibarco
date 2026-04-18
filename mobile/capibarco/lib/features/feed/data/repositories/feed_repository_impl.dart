import 'dart:convert';

import '../../../../core/cache/json_cache_store.dart';
import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/feed_models.dart';
import '../datasources/feed_remote_data_source.dart';
import '../dtos/feed_dto.dart';

class FeedRepositoryImpl {
  const FeedRepositoryImpl({
    required FeedRemoteDataSource remoteDataSource,
    required JsonCacheStore cacheStore,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _cacheStore = cacheStore,
       _errorMapper = errorMapper;

  final FeedRemoteDataSource _remoteDataSource;
  final JsonCacheStore _cacheStore;
  final ErrorMapper _errorMapper;

  Future<FeedPageEntity> getFeed({String pageToken = ''}) async {
    final cacheKey = 'feed:$pageToken';
    try {
      final remote = await _remoteDataSource.getFeed(pageToken: pageToken);
      await _cacheStore.write(
        cacheKey,
        jsonEncode(<String, dynamic>{
          'cards': remote.cards
              .map(
                (card) => <String, dynamic>{
                  'feed_card_id': card.id,
                  'feed_session_id': card.feedSessionId,
                  'animal': <String, dynamic>{
                    'animal_id': card.animalId,
                    'owner_profile_id': card.ownerProfileId,
                    'name': card.name,
                    'species': card.species,
                    'description': card.description,
                    'photos': <Map<String, dynamic>>[
                      <String, dynamic>{'url': card.photoUrl},
                    ],
                    'location': <String, dynamic>{'city': card.city},
                  },
                  'owner_display_name': card.ownerDisplayName,
                  'boosted': card.boosted,
                  'ranking_reasons': card.rankingReasons,
                },
              )
              .toList(),
          'next_page_token': remote.nextPageToken,
          'feed_session_id': remote.feedSessionId,
        }),
      );
      return remote.toDomain();
    } catch (error) {
      final mappedError = _errorMapper.map(error);
      final cachedRaw = _cacheStore.read(cacheKey);
      if (cachedRaw != null && mappedError.isRetryable) {
        final cached = FeedPageDto.fromJson(
          jsonDecode(cachedRaw) as Map<String, dynamic>,
          isStale: true,
        );
        return cached.toDomain();
      }
      throw mappedError;
    }
  }

  Future<SwipeOutcome> swipeAnimal({
    required String animalId,
    required String ownerProfileId,
    required bool liked,
    required String feedCardId,
    required String feedSessionId,
  }) async {
    final response = await _remoteDataSource.swipeAnimal(
      animalId: animalId,
      ownerProfileId: ownerProfileId,
      direction: liked ? 2 : 1,
      feedCardId: feedCardId,
      feedSessionId: feedSessionId,
    );
    return SwipeOutcome(
      matched: response['match'] != null,
      conversationId: response['conversation_id'] as String?,
    );
  }
}
