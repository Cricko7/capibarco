import '../../domain/entities/animal_listing.dart';

class AnimalListingDto {
  const AnimalListingDto({
    required this.id,
    required this.name,
    required this.status,
  });

  final String id;
  final String name;
  final String status;

  factory AnimalListingDto.fromJson(Map<String, dynamic> json) {
    final animal = json['animal'] as Map<String, dynamic>? ?? json;
    return AnimalListingDto(
      id: animal['animal_id'] as String? ?? '',
      name: animal['name'] as String? ?? '',
      status: animal['status'] as String? ?? 'ANIMAL_STATUS_UNSPECIFIED',
    );
  }

  AnimalListingEntity toDomain() {
    return AnimalListingEntity(id: id, name: name, status: status);
  }
}
