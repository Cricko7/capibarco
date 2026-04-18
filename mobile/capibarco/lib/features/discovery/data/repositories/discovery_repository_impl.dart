import 'dart:convert';

import '../../../../core/cache/json_cache_store.dart';
import '../../../../core/error/error_mapper.dart';
import '../../domain/entities/profile_summary.dart';
import '../datasources/discovery_remote_data_source.dart';
import '../dtos/profile_search_dto.dart';

class DiscoveryRepositoryImpl {
  const DiscoveryRepositoryImpl({
    required DiscoveryRemoteDataSource remoteDataSource,
    required JsonCacheStore cacheStore,
    required ErrorMapper errorMapper,
  }) : _remoteDataSource = remoteDataSource,
       _cacheStore = cacheStore,
       _errorMapper = errorMapper;

  final DiscoveryRemoteDataSource _remoteDataSource;
  final JsonCacheStore _cacheStore;
  final ErrorMapper _errorMapper;

  Future<ProfileSummaryPage> searchProfiles({
    required String query,
    required String city,
    String pageToken = '',
  }) async {
    final cacheKey = 'profiles:$query:$city:$pageToken';
    try {
      final remote = await _remoteDataSource.searchProfiles(
        query: query,
        city: city,
        pageToken: pageToken,
      );
      await _cacheStore.write(
        cacheKey,
        jsonEncode(<String, dynamic>{
          'profiles': remote.items
              .map(
                (profile) => <String, dynamic>{
                  'profile_id': profile.id,
                  'display_name': profile.displayName,
                  'bio': profile.bio,
                  'avatar_url': profile.avatarUrl,
                  'profile_type': profile.type,
                  'address': <String, dynamic>{'city': profile.city},
                  'reputation': <String, dynamic>{
                    'average_rating': profile.averageRating,
                    'reviews_count': profile.reviewsCount,
                  },
                },
              )
              .toList(),
          'page': <String, dynamic>{'next_page_token': remote.nextPageToken},
        }),
      );
      return remote.toDomain();
    } catch (error) {
      final cachedRaw = _cacheStore.read(cacheKey);
      if (cachedRaw != null) {
        final cached = ProfileSummaryPageDto.fromJson(
          jsonDecode(cachedRaw) as Map<String, dynamic>,
          isStale: true,
        );
        return cached.toDomain();
      }
      throw _errorMapper.map(error);
    }
  }
}
