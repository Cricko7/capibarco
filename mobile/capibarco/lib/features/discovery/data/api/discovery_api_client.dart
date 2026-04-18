import '../../../../core/network/rest_service_client.dart';
import '../dtos/profile_search_dto.dart';

class DiscoveryApiClient {
  const DiscoveryApiClient(this._client);

  final RestServiceClient _client;

  Future<ProfileSummaryPageDto> searchProfiles({
    required String query,
    required String city,
    String pageToken = '',
  }) async {
    final response = await _client.getMap(
      '/profiles',
      queryParameters: <String, dynamic>{
        if (query.isNotEmpty) 'query': query,
        if (city.isNotEmpty) 'city': city,
        'page_size': 20,
        if (pageToken.isNotEmpty) 'page_token': pageToken,
      },
    );
    return ProfileSummaryPageDto.fromJson(response);
  }
}
