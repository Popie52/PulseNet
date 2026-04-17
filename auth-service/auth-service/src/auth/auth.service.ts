import { BadRequestException, HttpException, HttpStatus, Injectable, UnauthorizedException } from "@nestjs/common";
import { LoginDto, SignupDto } from "./dto/auth.dto";
import * as bcrypt from "bcrypt";
import { JwtService } from "@nestjs/jwt";
import { InjectRepository } from "@nestjs/typeorm";
import { User } from "./entities/user.entity";
import { Repository } from "typeorm";

@Injectable()
export class AuthService {

    constructor(private jwtService: JwtService, @InjectRepository(User) private userRepo: Repository<User>) {}

    sayHello(): string {
        return "Hello World";
    }

    async refresh(refreshToken: string) {
        try {
            const payload = this.jwtService.verify(refreshToken);

            const user = await this.userRepo.findOneBy({ id: payload.sub });

            if(!user || !user.is_active) {
                throw new UnauthorizedException("Access Denied");
            }

            if(!user.refresh_token) {
                throw new UnauthorizedException("Access Denied");
            }

            const isMatch = await bcrypt.compare(refreshToken, user.refresh_token);

            if(!isMatch) {
                throw new UnauthorizedException("Access Denied");
            }

            const newAccessToken = this.jwtService.sign(
                {
                    sub: user.id,
                    email: user.email, 
                }, {
                    expiresIn:'15m'
                }
            )

            return {
                access_token: newAccessToken,
            };

        } catch(err) {
            throw new UnauthorizedException("Invalid refresh token");
        }
    }

    async createUser(email: string, username: string, password: string) {
        const existingUser = await this.userRepo.findOneBy({email});

        if(existingUser) {
            throw new BadRequestException("User already exists");
        }

        const hashedPassword = await bcrypt.hash(password, 10);

        try {
            const user = this.userRepo.create({
                email,
                username,
                password: hashedPassword,
            });

            const savedUser = await this.userRepo.save(user);

            return {
                id: savedUser.id,
                email: savedUser.email,
                username: savedUser.username,
            };
        } catch(error) {
            // Handle DB unique constraint (race condition)
            if(error.code == "23505") {
                throw new BadRequestException("User already exists");
            } 
            throw error;
        }
    }

    async signUp(data: SignupDto) {
        const user = await this.createUser(data.email, data.username ,data.password);
        return {
            message: "Signup successful",
            user,
        }
    }

    async login(data: LoginDto) {
        const {email, password} = data;

        const user = await this.userRepo.findOneBy({email});

        if(!user || !user.is_active) {
            throw new UnauthorizedException("Invalid Credentials");
        }

        const isMatch = await bcrypt.compare(password, user.password);
        
        if (!isMatch) {
            throw new UnauthorizedException("Invalid Credentials");
        }

        const payload = {
            sub: user.id,
            email: user.email,
        }

        // const token = this.jwtService.sign(payload);
        const accessToken = this.jwtService.sign(payload, {
            expiresIn: '15m', 
        });
        
        const refreshToken = this.jwtService.sign(payload, {
            expiresIn: '7d'
        });

        const hashedRefreshToken = await bcrypt.hash(refreshToken, 10);

        user.refresh_token = hashedRefreshToken;
        await this.userRepo.save(user);

        return {
            accessToken,
            refreshToken
        }
    }

    async logout(userId: string) {
         const user = await this.userRepo.findOneBy({id: userId});
         if(user) {
            user.refresh_token = null;
            await this.userRepo.save(user);
        }
        return { message: "Logged out successfully" };
    }
}