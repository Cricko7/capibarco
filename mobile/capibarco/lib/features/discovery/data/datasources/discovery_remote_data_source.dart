import '../api/discovery_api_client.dart';
import '../dtos/profile_search_dto.dart';

class DiscoveryRemoteDataSource {
  const DiscoveryRemoteDataSource(this._apiClient);

  final DiscoveryApiClient _apiClient;

  Future<ProfileSummaryPageDto> searchProfiles({
    required String query,
    required String city,
    String pageToken = '',
  }) =>
      _apiClient.searchProfiles(query: query, city: city, pageToken: pageToken);
}
