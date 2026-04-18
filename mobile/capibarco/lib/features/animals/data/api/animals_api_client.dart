import '../../../../core/network/rest_service_client.dart';
import '../dtos/animal_listing_dto.dart';

class AnimalsApiClient {
  const AnimalsApiClient(this._client);

  final RestServiceClient _client;

  Future<AnimalListingDto> createAnimal({
    required String ownerProfileId,
    required String ownerType,
    required String name,
    required String species,
    required String breed,
    required String sex,
    required String size,
    required int ageMonths,
    required String description,
    required List<String> traits,
    required bool vaccinated,
    required bool sterilized,
    required String status,
    required String visibility,
    required String city,
  }) async {
    final response = await _client.postJson(
      '/animals',
      idempotent: true,
      data: <String, dynamic>{
        'owner_profile_id': ownerProfileId,
        'owner_type': ownerType,
        'name': name,
        'species': species,
        'breed': breed,
        'sex': sex,
        'size': size,
        'age_months': ageMonths,
        'description': description,
        'traits': traits,
        'vaccinated': vaccinated,
        'sterilized': sterilized,
        'status': status,
        'visibility': visibility,
        'location': <String, dynamic>{'city': city},
      },
    );
    return AnimalListingDto.fromJson(response);
  }
}
