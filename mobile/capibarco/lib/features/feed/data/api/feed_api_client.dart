import '../../../../core/network/rest_service_client.dart';
import '../dtos/feed_dto.dart';

class FeedApiClient {
  const FeedApiClient(this._client);

  final RestServiceClient _client;

  Future<FeedPageDto> getFeed({
    String pageToken = '',
    int pageSize = 10,
  }) async {
    final response = await _client.getMap(
      '/feed',
      queryParameters: <String, dynamic>{
        'surface': 1,
        'page_size': pageSize,
        if (pageToken.isNotEmpty) 'page_token': pageToken,
      },
    );
    return FeedPageDto.fromJson(response);
  }

  Future<Map<String, dynamic>> swipeAnimal({
    required String animalId,
    required String ownerProfileId,
    required int direction,
    required String feedCardId,
    required String feedSessionId,
  }) {
    return _client.postJson(
      '/animals/$animalId/swipe',
      idempotent: true,
      data: <String, dynamic>{
        'owner_profile_id': ownerProfileId,
        'direction': direction,
        'feed_card_id': feedCardId,
        'feed_session_id': feedSessionId,
      },
    );
  }
}
