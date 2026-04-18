import '../api/feed_api_client.dart';
import '../dtos/feed_dto.dart';

class FeedRemoteDataSource {
  const FeedRemoteDataSource(this._apiClient);

  final FeedApiClient _apiClient;

  Future<FeedPageDto> getFeed({String pageToken = '', int pageSize = 10}) =>
      _apiClient.getFeed(pageToken: pageToken, pageSize: pageSize);

  Future<Map<String, dynamic>> swipeAnimal({
    required String animalId,
    required String ownerProfileId,
    required int direction,
    required String feedCardId,
    required String feedSessionId,
  }) {
    return _apiClient.swipeAnimal(
      animalId: animalId,
      ownerProfileId: ownerProfileId,
      direction: direction,
      feedCardId: feedCardId,
      feedSessionId: feedSessionId,
    );
  }
}
