import 'dart:typed_data';

import '../api/animals_api_client.dart';
import '../dtos/animal_editor_dto.dart';
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
    String? idempotencyKey,
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
    idempotencyKey: idempotencyKey,
  );

  Future<AnimalEditorDto> getAnimal({required String animalId}) =>
      _apiClient.getAnimal(animalId: animalId);

  Future<AnimalListingDto> updateAnimal({
    required String animalId,
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
    required String city,
  }) => _apiClient.updateAnimal(
    animalId: animalId,
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

  Future<AnimalListingDto> publishAnimal({required String animalId}) =>
      _apiClient.publishAnimal(animalId: animalId);
}
