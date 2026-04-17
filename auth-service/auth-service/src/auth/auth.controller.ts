import { Body, Controller, Get, Post } from "@nestjs/common";
import { AuthService } from "./auth.service";
import { AuthDto } from "./dto/auth.dto";

@Controller("/auth")
export class AuthController {
    constructor(private readonly authService: AuthService) {}

    @Get()
    sayHello(): string {
        return this.authService.sayHello();
    }

    @Post("/signup")
    signUp(@Body() bodyMessage: AuthDto) {
        return this.authService.signUp(bodyMessage);
    }
}