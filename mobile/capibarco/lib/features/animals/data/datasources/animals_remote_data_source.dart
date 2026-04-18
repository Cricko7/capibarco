import 'dart:typed_data';

import '../api/animals_api_client.dart';
import '../dtos/animal_listing_dto.dart';

class AnimalsRemoteDataSource {
  const AnimalsRemoteDataSource(this._apiClient);

  final AnimalsApiClient _apiClient;

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
  }) => _apiClient.createAnimal(
    ownerProfileId: ownerProfileId,
    ownerType: ownerType,
    name: name,
    species: species,
    breed: breed,
    sex: sex,
    size: size,
    ageMonths: ageMonths,
    description: description,
    traits: traits,
    vaccinated: vaccinated,
    sterilized: sterilized,
    status: status,
    visibility: visibility,
    city: city,
  );

  Future<AnimalListingDto> uploadAnimalPhoto({
    required String animalId,
    required Uint8List photoBytes,
    required String fileName,
  }) => _apiClient.uploadAnimalPhoto(
    animalId: animalId,
    photoBytes: photoBytes,
    fileName: fileName,
  );
}
