import 'dart:typed_data';

import '../../../../core/network/rest_service_client.dart';
import 'package:dio/dio.dart';
import '../dtos/animal_editor_dto.dart';
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
    String? idempotencyKey,
  }) async {
    final response = await _client.postJson(
      '/animals',
      idempotent: true,
      idempotencyKey: idempotencyKey,
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

  Future<AnimalEditorDto> getAnimal({required String animalId}) async {
    final response = await _client.getMap('/animals/$animalId');
    return AnimalEditorDto.fromJson(response);
  }

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
  }) async {
    final response = await _client.patchJson(
      '/animals/$animalId',
      data: <String, dynamic>{
        'animal': <String, dynamic>{
          'animal_id': animalId,
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
          'location': <String, dynamic>{'city': city},
        },
        'update_mask': const <String>[
          'name',
          'species',
          'breed',
          'sex',
          'size',
          'age_months',
          'description',
          'traits',
          'vaccinated',
          'sterilized',
          'location',
        ],
      },
    );
    return AnimalListingDto.fromJson(response);
  }

  Future<AnimalListingDto> uploadAnimalPhoto({
    required String animalId,
    required Uint8List photoBytes,
    required String fileName,
  }) async {
    final response = await _client.postMultipart(
      '/animals/$animalId/photos',
      idempotent: true,
      data: FormData.fromMap(<String, dynamic>{
        'photo': MultipartFile.fromBytes(photoBytes, filename: fileName),
      }),
    );
    return AnimalListingDto.fromJson(response);
  }

  Future<AnimalListingDto> publishAnimal({required String animalId}) async {
    final response = await _client.postJson('/animals/$animalId/publish');
    return AnimalListingDto.fromJson(response);
  }
}
